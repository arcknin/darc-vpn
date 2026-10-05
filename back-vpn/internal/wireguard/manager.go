package wireguard

import (
	"context"
	"darc-vpn/back-vpn/internal/domain"
	"errors"
	"net"
	"strings"
	"sync"
	"sync/atomic"
)

type ManagedRepository interface {
	ManagedPeers(context.Context) ([]domain.ManagedPeer, error)
	SetPeerResult(context.Context, string, string, string) error
}
type Manager struct {
	Mu     sync.Mutex
	repo   ManagedRepository
	kernel *Kernel
	port   int
	legacy func(context.Context) ([]domain.LegacyPeer, error)
	ready  atomic.Bool
}

func NewManager(repo ManagedRepository, kernel *Kernel, port int) *Manager {
	return &Manager{repo: repo, kernel: kernel, port: port}
}
func (m *Manager) WithLegacy(fn func(context.Context) ([]domain.LegacyPeer, error)) *Manager {
	m.legacy = fn
	return m
}
func (m *Manager) Ready() bool { return m.ready.Load() }

// Sync must run under Mu when shared with HTTP mutations.
func (m *Manager) Sync(ctx context.Context) error {
	m.ready.Store(false)
	if _, err := m.kernel.ServerPublicKey(ctx); err != nil {
		return err
	}
	desired, err := m.repo.ManagedPeers(ctx)
	if err != nil {
		return errors.New("peer state unavailable")
	}
	var legacy []domain.LegacyPeer
	if m.legacy != nil {
		legacy, err = m.legacy(ctx)
		if err != nil {
			return errors.New("legacy state unavailable")
		}
	}
	actual, err := m.kernel.Peers(ctx)
	if err != nil {
		return err
	}
	known := map[string]bool{}
	for _, p := range desired {
		known[p.PublicKey] = true
	}
	for _, p := range legacy {
		if known[p.PublicKey] {
			return errors.New("legacy identity already managed")
		}
		known[p.PublicKey] = true
	}
	for key := range actual {
		if !known[key] {
			return errors.New("unexpected WireGuard peer")
		}
	}
	for _, p := range desired {
		if !p.Active {
			removed, err := m.kernel.RemovePeers(ctx, []string{p.PublicKey})
			if err != nil {
				return err
			}
			if len(removed) != 1 || !removed[0].Removed {
				return errors.New("peer removal unconfirmed")
			}
			continue
		}
		ips, err := addressIPs(p.Address)
		if err != nil {
			return err
		}
		if err = m.kernel.EnsurePeer(ctx, p.PublicKey, ips); err != nil {
			if saveErr := m.repo.SetPeerResult(ctx, p.ProfileID, "failed", "peer_configuration_failed"); saveErr != nil {
				return errors.New("peer state persistence failed")
			}
			return err
		}
		if p.State != "ready" || p.Error != "" {
			if err = m.repo.SetPeerResult(ctx, p.ProfileID, "ready", ""); err != nil {
				return errors.New("peer state persistence failed")
			}
		}
	}
	for _, p := range legacy {
		var ips []net.IPNet
		for _, value := range p.AllowedIPs {
			_, prefix, err := net.ParseCIDR(value)
			if err != nil {
				return errors.New("legacy address invalid")
			}
			ips = append(ips, *prefix)
		}
		if err = m.kernel.EnsureLegacy(ctx, p.PublicKey, ips, p.PresharedKey, p.Keepalive); err != nil {
			return err
		}
	}
	m.ready.Store(true)
	return nil
}
func addressIPs(address string) ([]net.IPNet, error) {
	var ips []net.IPNet
	for _, part := range strings.Split(address, ",") {
		ip, cidr, err := net.ParseCIDR(strings.TrimSpace(part))
		if err != nil {
			return nil, errors.New("invalid peer address")
		}
		ones, bits := cidr.Mask.Size()
		if ones != bits {
			return nil, errors.New("peer address must be a host route")
		}
		cidr.IP = ip
		ips = append(ips, *cidr)
	}
	if len(ips) != 2 {
		return nil, errors.New("dual stack peer required")
	}
	return ips, nil
}
func (m *Manager) RemovePeers(ctx context.Context, keys []string) ([]domain.WireGuardRemoval, error) {
	m.ready.Store(false)
	if m.legacy != nil {
		peers, err := m.legacy(ctx)
		if err != nil {
			return nil, errors.New("legacy state unavailable")
		}
		for _, peer := range peers {
			for _, key := range keys {
				if peer.PublicKey == key {
					return nil, errors.New("legacy peer cannot be revoked as an account")
				}
			}
		}
	}
	return m.kernel.RemovePeers(ctx, keys)
}
func (m *Manager) ConfirmPeersRemoved(ctx context.Context, keys []string) (bool, error) {
	desired, err := m.repo.ManagedPeers(ctx)
	if err != nil {
		return false, errors.New("peer state unavailable")
	}
	for _, p := range desired {
		for _, key := range keys {
			if p.PublicKey == key && (p.Active || p.State != "revoked") {
				return false, nil
			}
		}
	}
	return m.kernel.ConfirmPeersRemoved(ctx, keys)
}
func (m *Manager) DeliveryMetadata(ctx context.Context, p domain.ProfileRecord) (string, int, error) {
	if p.ProvisioningState != "ready" {
		return "", 0, domain.NewCodedError(domain.ErrProfileNotReady, "profile is not ready")
	}
	ips, err := addressIPs(p.Address)
	if err != nil {
		return "", 0, domain.NewCodedError(domain.ErrProfileNotReady, "profile is not ready")
	}
	actual, err := m.kernel.Peers(ctx)
	if err != nil || !sameIPs(actual[p.PeerPublicKey], ips) {
		return "", 0, domain.NewCodedError(domain.ErrProfileNotReady, "profile is not ready")
	}
	key, err := m.kernel.ServerPublicKey(ctx)
	if err != nil {
		return "", 0, domain.NewCodedError(domain.ErrProfileNotReady, "profile is not ready")
	}
	return key, m.port, nil
}
