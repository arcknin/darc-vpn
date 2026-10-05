package storage

import (
	"context"
	"darc-vpn/back-vpn/internal/domain"
	"database/sql"
	"fmt"
	"net/netip"
	"strings"
	"time"
)

func (s *Store) ConfigureManaged(ctx context.Context, ipv4, ipv6 string) error {
	p4, e4 := netip.ParsePrefix(ipv4)
	p6, e6 := netip.ParsePrefix(ipv6)
	if e4 != nil || e6 != nil || !p4.Addr().Is4() || !p6.Addr().Is6() || p4 != p4.Masked() || p6 != p6.Masked() || p4.Bits() < 16 || p4.Bits() > 29 || p6.Bits() != 64 {
		return fmt.Errorf("invalid managed address pools")
	}
	// Validate all legacy records before any reservation is written.
	rows, err := s.db.QueryContext(ctx, `SELECT p.id,p.address,a.status FROM profiles p JOIN accounts a ON a.id=p.account_id`)
	if err != nil {
		return err
	}
	type legacy struct{ id, v4, v6, state string }
	var values []legacy
	seen := map[string]bool{}
	for rows.Next() {
		var id, address, status string
		if err = rows.Scan(&id, &address, &status); err != nil {
			rows.Close()
			return err
		}
		ips, err := parsePair(address, p4, p6)
		if err != nil {
			rows.Close()
			return fmt.Errorf("legacy profile addresses incompatible with configured pools")
		}
		for _, ip := range ips {
			if seen[ip] {
				rows.Close()
				return fmt.Errorf("legacy profile addresses collide")
			}
			seen[ip] = true
		}
		state := "pending"
		if status != "active" {
			state = "revoked"
		}
		values = append(values, legacy{id, ips[0], ips[1], state})
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, p := range values {
		if _, err = tx.ExecContext(ctx, `INSERT INTO peer_intents(profile_id,ipv4,ipv6,state,updated_at) VALUES(?,?,?,?,?) ON CONFLICT(profile_id) DO NOTHING`, p.id, p.v4, p.v6, p.state, formatTime(time.Now())); err != nil {
			return err
		}
	}
	// Existing reservations must match persisted addresses and the configured pools.
	var mismatch int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM peer_intents i JOIN profiles p ON p.id=i.profile_id WHERE p.address != i.ipv4 || '/32, ' || i.ipv6 || '/128'`).Scan(&mismatch); err != nil {
		return err
	}
	if mismatch != 0 {
		return fmt.Errorf("stored reservation differs from profile")
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	s.pool4 = p4
	s.pool6 = p6
	s.managed = true
	return nil
}
func parsePair(address string, p4, p6 netip.Prefix) ([]string, error) {
	parts := strings.Split(address, ",")
	if len(parts) != 2 {
		return nil, fmt.Errorf("dual stack address required")
	}
	var result []string
	for n, part := range parts {
		p, err := netip.ParsePrefix(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		pool := p4
		bits := 32
		if n == 1 {
			pool = p6
			bits = 128
		}
		if p.Bits() != bits || !pool.Contains(p.Addr()) || p.Addr().Compare(pool.Addr().Next()) <= 0 {
			return nil, fmt.Errorf("invalid reserved address")
		}
		if n == 0 && !pool.Contains(p.Addr().Next()) {
			return nil, fmt.Errorf("broadcast address")
		}
		result = append(result, p.Addr().String())
	}
	return result, nil
}
func (s *Store) prepareManagedProfile(ctx context.Context, tx *sql.Tx, p *domain.ProfileRecord) error {
	if !s.managed {
		return nil
	}
	rows, err := tx.QueryContext(ctx, `SELECT ipv4,ipv6 FROM peer_intents`)
	if err != nil {
		return err
	}
	used := map[string]bool{}
	for rows.Next() {
		var ip, ip6 string
		if err = rows.Scan(&ip, &ip6); err != nil {
			rows.Close()
			return err
		}
		used[ip] = true
		used[ip6] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	reserved, err := s.legacyPrefixes(ctx, tx)
	if err != nil {
		return err
	}
	v4 := s.pool4.Addr().Next().Next()
	v6 := s.pool6.Addr().Next().Next()
	for s.pool4.Contains(v4) && s.pool4.Contains(v4.Next()) {
		blocked := false
		for _, prefix := range reserved {
			if prefix.Contains(v4) || prefix.Contains(v6) {
				blocked = true
				break
			}
		}
		if !blocked && !used[v4.String()] && !used[v6.String()] {
			p.Address = v4.String() + "/32, " + v6.String() + "/128"
			p.ProvisioningState = "pending"
			return nil
		}
		v4 = v4.Next()
		v6 = v6.Next()
	}
	return domain.NewCodedError(domain.ErrPoolExhausted, "address pool exhausted")
}
func (s *Store) insertIntent(ctx context.Context, tx *sql.Tx, p domain.ProfileRecord) error {
	if !s.managed {
		return nil
	}
	ips, err := parsePair(p.Address, s.pool4, s.pool6)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO peer_intents(profile_id,ipv4,ipv6,state,updated_at) VALUES(?,?,?,'pending',?)`, p.ID, ips[0], ips[1], formatTime(time.Now()))
	return err
}
func (s *Store) ManagedPeers(ctx context.Context) ([]domain.ManagedPeer, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT p.id,p.account_id,p.peer_public_key,p.address,i.state,i.error,i.revision,i.updated_at,a.status FROM peer_intents i JOIN profiles p ON p.id=i.profile_id JOIN accounts a ON a.id=p.account_id ORDER BY p.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.ManagedPeer
	for rows.Next() {
		var p domain.ManagedPeer
		var at, status string
		if err = rows.Scan(&p.ProfileID, &p.AccountID, &p.PublicKey, &p.Address, &p.State, &p.Error, &p.Revision, &at, &status); err != nil {
			return nil, err
		}
		p.Active = status == "active" && p.State != "revoked"
		p.UpdatedAt, err = parseTime(at)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}
func (s *Store) SetPeerResult(ctx context.Context, id, state, code string) error {
	if state != "ready" && state != "failed" {
		return fmt.Errorf("invalid peer result")
	}
	_, err := s.db.ExecContext(ctx, `UPDATE peer_intents SET state=?,error=?,revision=revision+1,updated_at=? WHERE profile_id=? AND state!='revoked' AND EXISTS(SELECT 1 FROM profiles p JOIN accounts a ON a.id=p.account_id WHERE p.id=profile_id AND a.status='active')`, state, code, formatTime(time.Now()), id)
	return err
}
func (s *Store) BlockedAccounts(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT account_id FROM revocation_operations WHERE status!='completed'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}
