package platform

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"darc-vpn/back-vpn/internal/domain"
)

// FixedClock is a deterministic clock for local tests.
type FixedClock struct {
	Value time.Time
}

func (c FixedClock) Now() time.Time { return c.Value }

// SequenceIDGenerator is a deterministic ID source for local tests.
type SequenceIDGenerator struct {
	mu     sync.Mutex
	Prefix string
	Next   int
}

func (g *SequenceIDGenerator) NewID() (string, error) {
	if g == nil {
		return "", errors.New("nil id generator")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Next++
	return fmt.Sprintf("%s-%d", g.Prefix, g.Next), nil
}

// MemoryTermsStore is a local fake for the acceptance-file boundary. It is
// deliberately not a production persistence implementation.
type MemoryTermsStore struct {
	mu           sync.Mutex
	CurrentTerms domain.TermsDocument
	Receipts     []domain.TermsAcceptanceReceipt
}

func (s *MemoryTermsStore) Current(context.Context) (domain.TermsDocument, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.CurrentTerms, nil
}

func (s *MemoryTermsStore) PersistAcceptance(_ context.Context, receipt domain.TermsAcceptanceReceipt) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Receipts = append(s.Receipts, receipt)
	return nil
}

func (s *MemoryTermsStore) HasAcceptance(_ context.Context, accountID, version, sha256 string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, receipt := range s.Receipts {
		if receipt.AccountID == accountID && receipt.Version == version && receipt.SHA256 == sha256 && receipt.Accepted {
			return true, nil
		}
	}
	return false, nil
}

// MemoryAccessMetadataSink captures only the approved access metadata in
// tests. It does not accept request bodies, tokens, destinations, or content.
type MemoryAccessMetadataSink struct {
	mu      sync.Mutex
	Records []domain.AccessMetadata
}

func (s *MemoryAccessMetadataSink) Record(_ context.Context, metadata domain.AccessMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Records = append(s.Records, metadata)
	return nil
}

// MemorySessionStore is a local fake for session-port tests.
type MemorySessionStore struct {
	mu       sync.Mutex
	Sessions map[string]domain.Session
}

func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{Sessions: make(map[string]domain.Session)}
}

func (s *MemorySessionStore) Issue(_ context.Context, accountID string, role domain.Role) (domain.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := fmt.Sprintf("session-%d", len(s.Sessions)+1)
	session := domain.Session{ID: id, AccountID: accountID, Role: role}
	s.Sessions[id] = session
	return session, nil
}

func (s *MemorySessionStore) Lookup(_ context.Context, sessionID string) (domain.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.Sessions[sessionID]
	if !ok {
		return domain.Session{}, domain.NewCodedError(domain.ErrNotAuthenticated, "session is not authenticated")
	}
	return session, nil
}

func (s *MemorySessionStore) Revoke(_ context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Sessions, sessionID)
	return nil
}

func (s *MemorySessionStore) RevokeAccount(_ context.Context, accountID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, session := range s.Sessions {
		if session.AccountID == accountID {
			delete(s.Sessions, id)
		}
	}
	return nil
}

// MemoryTransactionalStore is a deterministic local fake. It clones state
// before invoking the callback and commits only when the callback succeeds.
type MemoryTransactionalStore struct {
	mu       sync.Mutex
	Accounts map[string]domain.AccountRecord
	Profiles map[string]domain.ProfileRecord
}

func NewMemoryTransactionalStore() *MemoryTransactionalStore {
	return &MemoryTransactionalStore{
		Accounts: make(map[string]domain.AccountRecord),
		Profiles: make(map[string]domain.ProfileRecord),
	}
}

func (s *MemoryTransactionalStore) WithinTransaction(ctx context.Context, fn func(context.Context, domain.StoreTx) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	accounts := cloneAccounts(s.Accounts)
	profiles := cloneProfiles(s.Profiles)
	tx := &memoryStoreTx{accounts: accounts, profiles: profiles}
	if err := fn(ctx, tx); err != nil {
		return err
	}
	s.Accounts = accounts
	s.Profiles = profiles
	return nil
}

type memoryStoreTx struct {
	accounts map[string]domain.AccountRecord
	profiles map[string]domain.ProfileRecord
}

func (tx *memoryStoreTx) Account(_ context.Context, accountID string) (domain.AccountRecord, error) {
	account, ok := tx.accounts[accountID]
	if !ok {
		return domain.AccountRecord{}, errors.New("account not found")
	}
	return account, nil
}

func (tx *memoryStoreTx) AccountByUsername(_ context.Context, username string) (domain.AccountRecord, error) {
	for _, account := range tx.accounts {
		if account.Username == username {
			return account, nil
		}
	}
	return domain.AccountRecord{}, errors.New("account not found")
}

func (tx *memoryStoreTx) SaveAccount(_ context.Context, account domain.AccountRecord) error {
	tx.accounts[account.ID] = account
	return nil
}

func (tx *memoryStoreTx) Profiles(_ context.Context, accountID string) ([]domain.ProfileRecord, error) {
	profiles := make([]domain.ProfileRecord, 0)
	for _, profile := range tx.profiles {
		if profile.AccountID == accountID {
			profiles = append(profiles, cloneProfile(profile))
		}
	}
	return profiles, nil
}

func (tx *memoryStoreTx) Profile(_ context.Context, accountID, profileID string) (domain.ProfileRecord, error) {
	profile, ok := tx.profiles[profileID]
	if !ok || profile.AccountID != accountID {
		return domain.ProfileRecord{}, errors.New("profile not found")
	}
	return cloneProfile(profile), nil
}

func (tx *memoryStoreTx) SaveProfile(_ context.Context, profile domain.ProfileRecord) error {
	tx.profiles[profile.ID] = cloneProfile(profile)
	return nil
}

func cloneAccounts(source map[string]domain.AccountRecord) map[string]domain.AccountRecord {
	copyOf := make(map[string]domain.AccountRecord, len(source))
	for key, value := range source {
		copyOf[key] = value
	}
	return copyOf
}

func cloneProfiles(source map[string]domain.ProfileRecord) map[string]domain.ProfileRecord {
	copyOf := make(map[string]domain.ProfileRecord, len(source))
	for key, value := range source {
		copyOf[key] = cloneProfile(value)
	}
	return copyOf
}

func cloneProfile(profile domain.ProfileRecord) domain.ProfileRecord {
	profile.AllowedIPs = append([]string(nil), profile.AllowedIPs...)
	profile.DNS = append([]string(nil), profile.DNS...)
	profile.EncryptedPrivate.Ciphertext = append([]byte(nil), profile.EncryptedPrivate.Ciphertext...)
	profile.EncryptedPrivate.Nonce = append([]byte(nil), profile.EncryptedPrivate.Nonce...)
	return profile
}

// DeterministicKeyGenerator is only a local fake. It must never be used for
// real WireGuard credentials.
type DeterministicKeyGenerator struct {
	mu   sync.Mutex
	Next int
}

func (g *DeterministicKeyGenerator) Generate(context.Context) (domain.WireGuardKeyPair, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Next++
	return domain.WireGuardKeyPair{
		PrivateKey: fmt.Sprintf("test-private-key-%d", g.Next),
		PublicKey:  fmt.Sprintf("test-public-key-%d", g.Next),
	}, nil
}

// FakeWireGuardController models only the local control-port protocol.
type FakeWireGuardController struct {
	mu              sync.Mutex
	RemovalResponse []domain.WireGuardRemoval
	Confirmed       bool
	ConfirmationErr error
	RemoveErr       error
	Requests        [][]string
}

func (c *FakeWireGuardController) RemovePeers(_ context.Context, peerIDs []string) ([]domain.WireGuardRemoval, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Requests = append(c.Requests, append([]string(nil), peerIDs...))
	if c.RemoveErr != nil {
		return nil, c.RemoveErr
	}
	return append([]domain.WireGuardRemoval(nil), c.RemovalResponse...), nil
}

func (c *FakeWireGuardController) ConfirmPeersRemoved(_ context.Context, _ []string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ConfirmationErr != nil {
		return false, c.ConfirmationErr
	}
	return c.Confirmed, nil
}
