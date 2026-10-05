package storage

import (
	"context"
	crypto "darc-vpn/back-vpn/internal/crypto"
	"darc-vpn/back-vpn/internal/domain"
	"database/sql"
	"encoding/json"
	"errors"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"net/netip"
	"reflect"
	"sort"
	"strings"
	"time"
)

func canonicalLegacy(p domain.LegacyPeer) (domain.LegacyPeer, error) {
	if key, err := wgtypes.ParseKey(p.PublicKey); err != nil || key == (wgtypes.Key{}) {
		return p, errors.New("invalid legacy peer")
	}
	if p.PresharedKey != "" {
		if _, err := wgtypes.ParseKey(p.PresharedKey); err != nil {
			return p, errors.New("invalid legacy PSK")
		}
	}
	if p.Keepalive < 0 || p.Keepalive > 65535 || len(p.AllowedIPs) == 0 {
		return p, errors.New("invalid legacy peer settings")
	}
	seen := map[string]bool{}
	var ips []string
	for _, value := range p.AllowedIPs {
		ip, err := netip.ParsePrefix(value)
		if err != nil || ip.Addr().Is4In6() {
			return p, errors.New("invalid legacy prefix")
		}
		ip = ip.Masked()
		if ip.Bits() == 0 {
			return p, errors.New("legacy default routes require separate routing review")
		}
		if !seen[ip.String()] {
			ips = append(ips, ip.String())
			seen[ip.String()] = true
		}
	}
	sort.Strings(ips)
	p.AllowedIPs = ips
	return p, nil
}
func legacyAAD(key string) []byte {
	return crypto.AssociatedData("legacy-wireguard-psk", key, crypto.Version)
}
func (s *Store) ImportLegacy(ctx context.Context, peers []domain.LegacyPeer, cipher domain.ProfileCipher) error {
	if cipher == nil || len(peers) == 0 {
		return errors.New("legacy import input required")
	}
	seen := map[string]bool{}
	var canonical []domain.LegacyPeer
	for _, p := range peers {
		p, err := canonicalLegacy(p)
		if err != nil {
			return err
		}
		if seen[p.PublicKey] {
			return errors.New("duplicate legacy peer")
		}
		seen[p.PublicKey] = true
		canonical = append(canonical, p)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	existing, err := readLegacy(ctx, tx, cipher)
	if err != nil {
		return err
	}
	combined := map[string]domain.LegacyPeer{}
	for _, p := range existing {
		combined[p.PublicKey] = p
	}
	for _, p := range canonical {
		if old, ok := combined[p.PublicKey]; ok && !reflect.DeepEqual(old, p) {
			return errors.New("legacy peer conflict")
		}
		combined[p.PublicKey] = p
	}
	var prefixes []struct {
		key string
		p   netip.Prefix
	}
	for _, peer := range combined {
		for _, value := range peer.AllowedIPs {
			prefix, _ := netip.ParsePrefix(value)
			for _, old := range prefixes {
				if old.key != peer.PublicKey && old.p.Overlaps(prefix) {
					return errors.New("legacy prefixes overlap")
				}
			}
			prefixes = append(prefixes, struct {
				key string
				p   netip.Prefix
			}{peer.PublicKey, prefix})
		}
	}
	rows, err := tx.QueryContext(ctx, `SELECT peer_public_key,address FROM profiles`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var key, address string
		if err = rows.Scan(&key, &address); err != nil {
			rows.Close()
			return err
		}
		if seen[key] {
			rows.Close()
			return errors.New("legacy key already managed")
		}
		for _, value := range splitAddresses(address) {
			prefix, err := netip.ParsePrefix(value)
			if err != nil {
				rows.Close()
				return errors.New("managed address invalid")
			}
			for _, old := range prefixes {
				if old.p.Overlaps(prefix) {
					rows.Close()
					return errors.New("legacy and managed addresses overlap")
				}
			}
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, p := range canonical {
		var exists int
		err = tx.QueryRowContext(ctx, `SELECT 1 FROM legacy_peers WHERE public_key=?`, p.PublicKey).Scan(&exists)
		if err == nil {
			continue
		}
		if err != sql.ErrNoRows {
			return err
		}
		encrypted, err := cipher.Encrypt(ctx, []byte(p.PresharedKey), legacyAAD(p.PublicKey))
		if err != nil {
			return errors.New("legacy encryption failed")
		}
		ips, _ := json.Marshal(p.AllowedIPs)
		if _, err = tx.ExecContext(ctx, `INSERT INTO legacy_peers(public_key,allowed_ips_json,psk_ciphertext,psk_nonce,encryption_version,keepalive,imported_at) VALUES(?,?,?,?,?,?,?)`, p.PublicKey, string(ips), encrypted.Ciphertext, encrypted.Nonce, encrypted.Version, p.Keepalive, formatTime(time.Now())); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type legacyReader interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func readLegacy(ctx context.Context, reader legacyReader, cipher domain.ProfileCipher) ([]domain.LegacyPeer, error) {
	rows, err := reader.QueryContext(ctx, `SELECT public_key,allowed_ips_json,psk_ciphertext,psk_nonce,encryption_version,keepalive FROM legacy_peers ORDER BY public_key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.LegacyPeer
	for rows.Next() {
		var p domain.LegacyPeer
		var ips string
		var encrypted domain.EncryptedPrivateProfile
		if err = rows.Scan(&p.PublicKey, &ips, &encrypted.Ciphertext, &encrypted.Nonce, &encrypted.Version, &p.Keepalive); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(ips), &p.AllowedIPs); err != nil {
			return nil, errors.New("legacy state invalid")
		}
		secret, err := cipher.Decrypt(ctx, encrypted, legacyAAD(p.PublicKey))
		if err != nil {
			return nil, errors.New("legacy PSK recovery failed")
		}
		p.PresharedKey = string(secret)
		p, err = canonicalLegacy(p)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}
func (s *Store) LegacyPeers(ctx context.Context, cipher domain.ProfileCipher) ([]domain.LegacyPeer, error) {
	return readLegacy(ctx, s.db, cipher)
}
func (s *Store) legacyPrefixes(ctx context.Context, tx *sql.Tx) ([]netip.Prefix, error) {
	rows, err := tx.QueryContext(ctx, `SELECT allowed_ips_json FROM legacy_peers`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []netip.Prefix
	for rows.Next() {
		var raw string
		var values []string
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(raw), &values); err != nil {
			return nil, errors.New("invalid legacy reservation")
		}
		for _, value := range values {
			p, err := netip.ParsePrefix(value)
			if err != nil {
				return nil, errors.New("invalid legacy reservation")
			}
			result = append(result, p)
		}
	}
	return result, rows.Err()
}

func splitAddresses(s string) []string {
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
