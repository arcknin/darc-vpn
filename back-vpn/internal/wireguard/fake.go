package wireguard

import (
	"context"
	"errors"
	"sync"

	"darc-vpn/back-vpn/internal/domain"
)

type FakeMode string

const (
	FakeConfirmAll       FakeMode = "confirm_all"
	FakeFailure          FakeMode = "failure"
	FakeOmitConfirmation FakeMode = "omit_confirmation"
	FakePartial          FakeMode = "partial"
)

// Fake is a deterministic local controller. It exercises the revocation
// protocol without claiming any effect on a WireGuard server.
type Fake struct {
	mu           sync.Mutex
	mode         FakeMode
	removeCalls  [][]string
	confirmCalls [][]string
}

func NewFake(mode FakeMode) *Fake {
	return &Fake{mode: mode}
}

func (f *Fake) RemovePeers(_ context.Context, peerIDs []string) ([]domain.WireGuardRemoval, error) {
	f.mu.Lock()
	f.removeCalls = append(f.removeCalls, append([]string(nil), peerIDs...))
	mode := f.mode
	f.mu.Unlock()

	if mode == FakeFailure {
		return nil, errors.New("simulated WireGuard removal failure")
	}
	limit := len(peerIDs)
	if mode == FakePartial && limit > 0 {
		limit--
	}
	result := make([]domain.WireGuardRemoval, 0, limit)
	for _, peerID := range peerIDs[:limit] {
		result = append(result, domain.WireGuardRemoval{PeerID: peerID, Removed: true})
	}
	return result, nil
}

func (f *Fake) ConfirmPeersRemoved(_ context.Context, peerIDs []string) (bool, error) {
	f.mu.Lock()
	f.confirmCalls = append(f.confirmCalls, append([]string(nil), peerIDs...))
	mode := f.mode
	f.mu.Unlock()
	return mode == FakeConfirmAll, nil
}

func (f *Fake) RemoveCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.removeCalls)
}

func (f *Fake) RemovalCalls() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([][]string, len(f.removeCalls))
	for index, call := range f.removeCalls {
		result[index] = append([]string(nil), call...)
	}
	return result
}
