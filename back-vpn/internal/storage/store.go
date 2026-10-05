package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"darc-vpn/back-vpn/internal/auth"
	"darc-vpn/back-vpn/internal/domain"
	"darc-vpn/back-vpn/migrations"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db      *sql.DB
	pool4   netip.Prefix
	pool6   netip.Prefix
	managed bool
}

func Open(path string) (*Store, error) {
	dsn := (&url.URL{Scheme: "file", Path: path, RawQuery: "_foreign_keys=on&_busy_timeout=5000"}).String()
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open SQLite: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping SQLite: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) DB() *sql.DB { return s.db }

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Migrate(ctx context.Context) error { return migrations.Apply(ctx, s.db) }

func (s *Store) CreateAdmin(ctx context.Context, account domain.AccountRecord, passwordHash string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO administrators(id, username, password_hash, status, created_at) VALUES (?, ?, ?, ?, ?)`,
		account.ID, account.Username, passwordHash, account.Status, formatTime(account.CreatedAt))
	return mapConstraint(err)
}

func (s *Store) AdminExists(ctx context.Context, username string) (bool, error) {
	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM administrators WHERE username = ?`, username).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (s *Store) CreateAccountWithPrimary(ctx context.Context, account domain.AccountRecord, passwordHash string, profile domain.ProfileRecord) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO accounts(id, username, password_hash, status, created_at, primary_profile_id) VALUES (?, ?, ?, ?, ?, ?)`,
		account.ID, account.Username, passwordHash, account.Status, formatTime(account.CreatedAt), account.PrimaryID); err != nil {
		return mapConstraint(err)
	}
	if err := s.prepareManagedProfile(ctx, tx, &profile); err != nil {
		return err
	}
	if err := insertProfile(ctx, tx, profile); err != nil {
		return err
	}
	if err := s.insertIntent(ctx, tx, profile); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) AddProfile(ctx context.Context, accountID string, profile domain.ProfileRecord) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM accounts WHERE id = ?`, accountID).Scan(&status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NewCodedError(domain.ErrProfileNotFound, "account was not found")
		}
		return err
	}
	if status != string(domain.AccountActive) {
		return domain.NewCodedError(domain.ErrAccountBlocked, "account is blocked")
	}
	if err := s.prepareManagedProfile(ctx, tx, &profile); err != nil {
		return err
	}
	if err := insertProfile(ctx, tx, profile); err != nil {
		return err
	}
	if err := s.insertIntent(ctx, tx, profile); err != nil {
		return err
	}
	return tx.Commit()
}

func insertProfile(ctx context.Context, executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, profile domain.ProfileRecord) error {
	allowed, err := json.Marshal(profile.AllowedIPs)
	if err != nil {
		return err
	}
	dns, err := json.Marshal(profile.DNS)
	if err != nil {
		return err
	}
	_, err = executor.ExecContext(ctx, `INSERT INTO profiles(
		id, account_id, display_name, is_primary, endpoint, peer_public_key, address,
		allowed_ips_json, dns_json, private_ciphertext, private_nonce, encryption_version, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		profile.ID, profile.AccountID, profile.DisplayName, profile.IsPrimary, profile.Endpoint, profile.PeerPublicKey,
		profile.Address, string(allowed), string(dns), profile.EncryptedPrivate.Ciphertext, profile.EncryptedPrivate.Nonce,
		profile.EncryptedPrivate.Version, formatTime(profile.CreatedAt))
	return mapConstraint(err)
}

func (s *Store) CredentialByUsername(ctx context.Context, username string, role domain.Role) (auth.Credential, error) {
	var query string
	switch role {
	case domain.RoleFriend:
		query = `SELECT id, password_hash, status FROM accounts WHERE username = ?`
	case domain.RoleAdmin:
		query = `SELECT id, password_hash, status FROM administrators WHERE username = ?`
	default:
		return auth.Credential{}, auth.ErrCredentialNotFound
	}
	var credential auth.Credential
	credential.Role = role
	if err := s.db.QueryRowContext(ctx, query, username).Scan(&credential.AccountID, &credential.PasswordHash, &credential.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return auth.Credential{}, auth.ErrCredentialNotFound
		}
		return auth.Credential{}, err
	}
	return credential, nil
}

func (s *Store) SaveSession(ctx context.Context, session auth.StoredSession) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO sessions(id, token_digest, account_id, role, expires_at, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		session.ID, session.TokenDigest, session.AccountID, session.Role, formatTime(session.ExpiresAt), formatTime(session.CreatedAt))
	return err
}

func (s *Store) SessionByDigest(ctx context.Context, digest []byte) (auth.StoredSession, error) {
	var session auth.StoredSession
	var expiresAt, createdAt, status string
	row := s.db.QueryRowContext(ctx, `
		SELECT s.id, s.token_digest, s.account_id, s.role, s.expires_at, s.created_at,
		       CASE s.role WHEN 'friend' THEN a.status ELSE ad.status END
		FROM sessions s
		LEFT JOIN accounts a ON s.role = 'friend' AND a.id = s.account_id
		LEFT JOIN administrators ad ON s.role = 'admin' AND ad.id = s.account_id
		WHERE s.token_digest = ?`, digest)
	if err := row.Scan(&session.ID, &session.TokenDigest, &session.AccountID, &session.Role, &expiresAt, &createdAt, &status); err != nil {
		return auth.StoredSession{}, err
	}
	var err error
	session.ExpiresAt, err = parseTime(expiresAt)
	if err != nil {
		return auth.StoredSession{}, err
	}
	session.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return auth.StoredSession{}, err
	}
	session.Status = domain.AccountStatus(status)
	return session, nil
}

func (s *Store) DeleteSessionByDigest(ctx context.Context, digest []byte) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_digest = ?`, digest)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) SetAccountStatus(ctx context.Context, accountID string, status domain.AccountStatus) error {
	result, err := s.db.ExecContext(ctx, `UPDATE accounts SET status = ? WHERE id = ?`, status, accountID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// BeginAccountBlock atomically blocks the account, invalidates every session,
// and creates the immutable peer snapshot before any controller is called.
func (s *Store) BeginAccountBlock(ctx context.Context, accountID string, now time.Time) (domain.RevocationOperation, bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.RevocationOperation{}, false, err
	}
	defer tx.Rollback()

	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM accounts WHERE id = ?`, accountID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.RevocationOperation{}, false, domain.NewCodedError(domain.ErrProfileNotFound, "account was not found")
		}
		return domain.RevocationOperation{}, false, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE accounts SET status = ? WHERE id = ?`, domain.AccountBlocked, accountID); err != nil {
		return domain.RevocationOperation{}, false, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE account_id = ?`, accountID); err != nil {
		return domain.RevocationOperation{}, false, err
	}

	if _, err := tx.ExecContext(ctx, `UPDATE peer_intents SET state='revoked', error='', revision=revision+1, updated_at=? WHERE profile_id IN (SELECT id FROM profiles WHERE account_id=?) AND state!='revoked'`, formatTime(now), accountID); err != nil {
		return domain.RevocationOperation{}, false, err
	}
	existing, err := loadRevocation(ctx, tx, accountID)
	if err == nil {
		if err := tx.Commit(); err != nil {
			return domain.RevocationOperation{}, false, err
		}
		return existing, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domain.RevocationOperation{}, false, err
	}

	rows, err := tx.QueryContext(ctx, `SELECT id, peer_public_key FROM profiles WHERE account_id = ? ORDER BY is_primary DESC, created_at, id`, accountID)
	if err != nil {
		return domain.RevocationOperation{}, false, err
	}
	var peers []domain.PeerRevocation
	for rows.Next() {
		var peer domain.PeerRevocation
		if err := rows.Scan(&peer.ProfileID, &peer.PeerID); err != nil {
			_ = rows.Close()
			return domain.RevocationOperation{}, false, err
		}
		peers = append(peers, peer)
	}
	if err := rows.Close(); err != nil {
		return domain.RevocationOperation{}, false, err
	}
	if err := rows.Err(); err != nil {
		return domain.RevocationOperation{}, false, err
	}
	if len(peers) == 0 {
		return domain.RevocationOperation{}, false, domain.NewCodedError(domain.ErrRevocationNotConfirmed, "account has no peers to revoke")
	}

	now = now.UTC()
	if _, err := tx.ExecContext(ctx, `INSERT INTO revocation_operations(account_id, status, error, updated_at) VALUES (?, ?, '', ?)`, accountID, domain.RevocationPending, formatTime(now)); err != nil {
		return domain.RevocationOperation{}, false, err
	}
	for _, peer := range peers {
		if _, err := tx.ExecContext(ctx, `INSERT INTO revocation_peers(account_id, profile_id, peer_id, removed, error) VALUES (?, ?, ?, 0, '')`, accountID, peer.ProfileID, peer.PeerID); err != nil {
			return domain.RevocationOperation{}, false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.RevocationOperation{}, false, err
	}
	return domain.RevocationOperation{AccountID: accountID, Status: domain.RevocationPending, Peers: peers, UpdatedAt: now}, true, nil
}

func (s *Store) SaveRevocation(ctx context.Context, operation domain.RevocationOperation) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	existing, err := loadRevocation(ctx, tx, operation.AccountID)
	if err != nil {
		return err
	}
	if !samePeerSnapshot(existing.Peers, operation.Peers) {
		return errors.New("revocation peer snapshot cannot be changed")
	}
	for _, peer := range operation.Peers {
		result, err := tx.ExecContext(ctx, `UPDATE revocation_peers SET removed = ?, error = ? WHERE account_id = ? AND profile_id = ? AND peer_id = ?`,
			peer.Removed, peer.Error, operation.AccountID, peer.ProfileID, peer.PeerID)
		if err != nil {
			return err
		}
		if count, err := result.RowsAffected(); err != nil || count != 1 {
			if err != nil {
				return err
			}
			return errors.New("revocation peer snapshot is incomplete")
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE revocation_operations SET status = ?, error = ?, updated_at = ? WHERE account_id = ?`,
		operation.Status, operation.Error, formatTime(operation.UpdatedAt), operation.AccountID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Revocation(ctx context.Context, accountID string) (domain.RevocationOperation, error) {
	operation, err := loadRevocation(ctx, s.db, accountID)
	if err == nil {
		return operation, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domain.RevocationOperation{}, err
	}
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM accounts WHERE id = ?`, accountID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.RevocationOperation{}, domain.NewCodedError(domain.ErrProfileNotFound, "account was not found")
		}
		return domain.RevocationOperation{}, err
	}
	return domain.RevocationOperation{}, domain.NewCodedError(domain.ErrRevocationNotConfirmed, "revocation has not started")
}

type revocationQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func loadRevocation(ctx context.Context, queryer revocationQueryer, accountID string) (domain.RevocationOperation, error) {
	var operation domain.RevocationOperation
	var status, updatedAt string
	err := queryer.QueryRowContext(ctx, `SELECT account_id, status, error, updated_at FROM revocation_operations WHERE account_id = ?`, accountID).
		Scan(&operation.AccountID, &status, &operation.Error, &updatedAt)
	if err != nil {
		return domain.RevocationOperation{}, err
	}
	operation.Status = domain.RevocationStatus(status)
	operation.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return domain.RevocationOperation{}, err
	}
	rows, err := queryer.QueryContext(ctx, `
		SELECT rp.profile_id, rp.peer_id, rp.removed, rp.error
		FROM revocation_peers rp
		JOIN profiles p ON p.id = rp.profile_id AND p.account_id = rp.account_id
		WHERE rp.account_id = ?
		ORDER BY p.is_primary DESC, p.created_at, p.id`, accountID)
	if err != nil {
		return domain.RevocationOperation{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var peer domain.PeerRevocation
		if err := rows.Scan(&peer.ProfileID, &peer.PeerID, &peer.Removed, &peer.Error); err != nil {
			return domain.RevocationOperation{}, err
		}
		operation.Peers = append(operation.Peers, peer)
	}
	return operation, rows.Err()
}

func samePeerSnapshot(left, right []domain.PeerRevocation) bool {
	if len(left) != len(right) {
		return false
	}
	want := make(map[string]string, len(left))
	for _, peer := range left {
		want[peer.ProfileID] = peer.PeerID
	}
	for _, peer := range right {
		peerID, exists := want[peer.ProfileID]
		if !exists || peerID != peer.PeerID {
			return false
		}
		delete(want, peer.ProfileID)
	}
	return len(want) == 0
}

func (s *Store) PersistAcceptance(ctx context.Context, receipt domain.TermsAcceptanceReceipt) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO terms_acceptances(account_id, version, sha256, accepted_at, event_id) VALUES (?, ?, ?, ?, ?)`,
		receipt.AccountID, receipt.Version, receipt.SHA256, formatTime(receipt.AcceptedAt), receipt.EventID)
	return err
}

func (s *Store) HasAcceptance(ctx context.Context, accountID, version, sha256 string) (bool, error) {
	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM terms_acceptances WHERE account_id = ? AND version = ? AND sha256 = ?`, accountID, version, sha256).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (s *Store) Account(ctx context.Context, accountID string) (domain.AccountRecord, error) {
	var account domain.AccountRecord
	var createdAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT a.id, a.username, a.status, a.created_at, a.primary_profile_id, COUNT(p.id)
		FROM accounts a LEFT JOIN profiles p ON p.account_id = a.id
		WHERE a.id = ? GROUP BY a.id`, accountID).
		Scan(&account.ID, &account.Username, &account.Status, &createdAt, &account.PrimaryID, &account.ProfileCount)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.AccountRecord{}, domain.NewCodedError(domain.ErrProfileNotFound, "account was not found")
	}
	if err != nil {
		return domain.AccountRecord{}, err
	}
	account.Role = domain.RoleFriend
	account.CreatedAt, err = parseTime(createdAt)
	return account, err
}

func (s *Store) Accounts(ctx context.Context) ([]domain.AccountRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.id, a.username, a.status, a.created_at, a.primary_profile_id, COUNT(p.id)
		FROM accounts a LEFT JOIN profiles p ON p.account_id = a.id
		GROUP BY a.id ORDER BY a.created_at, a.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.AccountRecord
	for rows.Next() {
		var account domain.AccountRecord
		var createdAt string
		if err := rows.Scan(&account.ID, &account.Username, &account.Status, &createdAt, &account.PrimaryID, &account.ProfileCount); err != nil {
			return nil, err
		}
		account.Role = domain.RoleFriend
		account.CreatedAt, err = parseTime(createdAt)
		if err != nil {
			return nil, err
		}
		result = append(result, account)
	}
	return result, rows.Err()
}

func (s *Store) Profiles(ctx context.Context, accountID string) ([]domain.ProfileRecord, error) {
	rows, err := s.db.QueryContext(ctx, profileSelect+` WHERE account_id = ? ORDER BY is_primary DESC, created_at, id`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.ProfileRecord
	for rows.Next() {
		profile, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, profile)
	}
	return result, rows.Err()
}

func (s *Store) Profile(ctx context.Context, accountID, profileID string) (domain.ProfileRecord, error) {
	profile, err := scanProfile(s.db.QueryRowContext(ctx, profileSelect+` WHERE account_id = ? AND id = ?`, accountID, profileID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ProfileRecord{}, domain.NewCodedError(domain.ErrProfileNotFound, "profile was not found")
	}
	return profile, err
}

const profileSelect = `SELECT id, account_id, display_name, is_primary, endpoint, peer_public_key, address,
	allowed_ips_json, dns_json, private_ciphertext, private_nonce, encryption_version, created_at, COALESCE((SELECT state FROM peer_intents WHERE profile_id=profiles.id),''), COALESCE((SELECT error FROM peer_intents WHERE profile_id=profiles.id),'') FROM profiles`

type scanner interface{ Scan(...any) error }

func scanProfile(row scanner) (domain.ProfileRecord, error) {
	var profile domain.ProfileRecord
	var allowed, dns, createdAt string
	if err := row.Scan(&profile.ID, &profile.AccountID, &profile.DisplayName, &profile.IsPrimary, &profile.Endpoint,
		&profile.PeerPublicKey, &profile.Address, &allowed, &dns, &profile.EncryptedPrivate.Ciphertext,
		&profile.EncryptedPrivate.Nonce, &profile.EncryptedPrivate.Version, &createdAt, &profile.ProvisioningState, &profile.ProvisioningError); err != nil {
		return domain.ProfileRecord{}, err
	}
	if err := json.Unmarshal([]byte(allowed), &profile.AllowedIPs); err != nil {
		return domain.ProfileRecord{}, err
	}
	if err := json.Unmarshal([]byte(dns), &profile.DNS); err != nil {
		return domain.ProfileRecord{}, err
	}
	created, err := parseTime(createdAt)
	if err != nil {
		return domain.ProfileRecord{}, err
	}
	profile.CreatedAt = created
	return profile, nil
}

// WithinTransaction preserves the B01 domain port for later batches.
func (s *Store) WithinTransaction(ctx context.Context, fn func(context.Context, domain.StoreTx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(ctx, &domainTx{tx: tx}); err != nil {
		return err
	}
	return tx.Commit()
}

type domainTx struct{ tx *sql.Tx }

func (d *domainTx) Account(ctx context.Context, accountID string) (domain.AccountRecord, error) {
	var account domain.AccountRecord
	var createdAt string
	err := d.tx.QueryRowContext(ctx, `SELECT id, username, status, created_at, primary_profile_id FROM accounts WHERE id = ?`, accountID).
		Scan(&account.ID, &account.Username, &account.Status, &createdAt, &account.PrimaryID)
	if err != nil {
		return domain.AccountRecord{}, err
	}
	account.Role = domain.RoleFriend
	account.CreatedAt, err = parseTime(createdAt)
	return account, err
}

func (d *domainTx) AccountByUsername(ctx context.Context, username string) (domain.AccountRecord, error) {
	var account domain.AccountRecord
	var createdAt string
	err := d.tx.QueryRowContext(ctx, `SELECT id, username, status, created_at, primary_profile_id FROM accounts WHERE username = ?`, username).
		Scan(&account.ID, &account.Username, &account.Status, &createdAt, &account.PrimaryID)
	if err != nil {
		return domain.AccountRecord{}, err
	}
	account.Role = domain.RoleFriend
	account.CreatedAt, err = parseTime(createdAt)
	return account, err
}

func (d *domainTx) SaveAccount(ctx context.Context, account domain.AccountRecord) error {
	_, err := d.tx.ExecContext(ctx, `UPDATE accounts SET username = ?, status = ?, primary_profile_id = ? WHERE id = ?`, account.Username, account.Status, account.PrimaryID, account.ID)
	return err
}

func (d *domainTx) Profiles(ctx context.Context, accountID string) ([]domain.ProfileRecord, error) {
	rows, err := d.tx.QueryContext(ctx, profileSelect+` WHERE account_id = ? ORDER BY is_primary DESC, created_at, id`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var profiles []domain.ProfileRecord
	for rows.Next() {
		profile, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}
	return profiles, rows.Err()
}

func (d *domainTx) Profile(ctx context.Context, accountID, profileID string) (domain.ProfileRecord, error) {
	return scanProfile(d.tx.QueryRowContext(ctx, profileSelect+` WHERE account_id = ? AND id = ?`, accountID, profileID))
}

func (d *domainTx) SaveProfile(ctx context.Context, profile domain.ProfileRecord) error {
	return insertProfile(ctx, d.tx, profile)
}

func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

func parseTime(value string) (time.Time, error) { return time.Parse(time.RFC3339Nano, value) }

func mapConstraint(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return domain.NewCodedError(domain.ErrInvalidRequest, "resource already exists")
	}
	return err
}
