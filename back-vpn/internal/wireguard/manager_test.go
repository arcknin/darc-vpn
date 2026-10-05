package wireguard

import (
	"context"
	"darc-vpn/back-vpn/internal/domain"
	"errors"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"net"
	"testing"
)

type peerRepo struct {
	records []domain.ManagedPeer
	fail    bool
}

func (r *peerRepo) ManagedPeers(context.Context) ([]domain.ManagedPeer, error) {
	if r.fail {
		return nil, errors.New("db failed")
	}
	return r.records, nil
}
func (r *peerRepo) SetPeerResult(_ context.Context, id, state, code string) error {
	if r.fail {
		return errors.New("db failed")
	}
	for i := range r.records {
		if r.records[i].ProfileID == id {
			r.records[i].State = state
			r.records[i].Error = code
		}
	}
	return nil
}
func TestManagerRecoversWithoutReinstallingRevokedAndRejectsUnknown(t *testing.T) {
	c, _ := wgtypes.GeneratePrivateKey()
	s, _ := wgtypes.GeneratePrivateKey()
	stub := &deviceStub{device: &wgtypes.Device{PublicKey: s.PublicKey()}}
	k, _ := NewKernelWithClient("wg0", stub)
	repo := &peerRepo{records: []domain.ManagedPeer{{ProfileID: "p", PublicKey: c.PublicKey().String(), Address: "10.77.77.2/32, fd42:77:77::2/128", State: "pending", Active: true}}}
	m := NewManager(repo, k, 51443)
	if err := m.Sync(context.Background()); err != nil || !m.Ready() {
		t.Fatalf("sync: %v", err)
	}
	if repo.records[0].State != "ready" {
		t.Fatal("not persisted ready")
	}
	stub.device.Peers = nil
	if err := m.Sync(context.Background()); err != nil || len(stub.device.Peers) != 1 {
		t.Fatal("did not recover missing peer")
	}
	repo.records[0].Active = false
	repo.records[0].State = "revoked"
	if err := m.Sync(context.Background()); err != nil || len(stub.device.Peers) != 0 {
		t.Fatal("revocation not reconciled")
	}
	if err := m.Sync(context.Background()); err != nil || len(stub.device.Peers) != 0 {
		t.Fatal("revoked peer reappeared")
	}
	u, _ := wgtypes.GeneratePrivateKey()
	stub.device.Peers = []wgtypes.Peer{{PublicKey: u.PublicKey(), AllowedIPs: []net.IPNet{}}}
	if err := m.Sync(context.Background()); err == nil || m.Ready() {
		t.Fatal("unknown peer accepted")
	}
	if len(stub.device.Peers) != 1 {
		t.Fatal("unknown peer removed")
	}
	stub.device.Peers = nil
	repo.records[0].Active = true
	repo.records[0].State = "pending"
	stub.lie = true
	if err := m.Sync(context.Background()); err == nil || m.Ready() || repo.records[0].State != "failed" {
		t.Fatal("lying device marked ready")
	}
}
func TestLegacySurvivesManagedRevocationAndColdReconcile(t *testing.T) {
	ctx := context.Background()
	managed, _ := wgtypes.GeneratePrivateKey()
	legacy, _ := wgtypes.GeneratePrivateKey()
	psk, _ := wgtypes.GenerateKey()
	server, _ := wgtypes.GeneratePrivateKey()
	stub := &deviceStub{device: &wgtypes.Device{PublicKey: server.PublicKey()}}
	kernel, _ := NewKernelWithClient("wg0", stub)
	repo := &peerRepo{records: []domain.ManagedPeer{{ProfileID: "p", PublicKey: managed.PublicKey().String(), Address: "10.77.77.2/32, fd42:77:77::2/128", State: "pending", Active: true}}}
	m := NewManager(repo, kernel, 51442).WithLegacy(func(context.Context) ([]domain.LegacyPeer, error) {
		return []domain.LegacyPeer{{PublicKey: legacy.PublicKey().String(), AllowedIPs: []string{"10.77.77.42/32"}, PresharedKey: psk.String(), Keepalive: 25}}, nil
	})
	if err := m.Sync(ctx); err != nil || len(stub.device.Peers) != 2 {
		t.Fatalf("coexistence failed %v", err)
	}
	repo.records[0].Active = false
	repo.records[0].State = "revoked"
	if err := m.Sync(ctx); err != nil || len(stub.device.Peers) != 1 || stub.device.Peers[0].PresharedKey != psk {
		t.Fatal("managed block damaged legacy")
	}
	stub.device.Peers = nil
	if err := m.Sync(ctx); err != nil || len(stub.device.Peers) != 1 || stub.device.Peers[0].PresharedKey != psk {
		t.Fatal("cold restore lost PSK")
	}
	if _, err := m.RemovePeers(ctx, []string{legacy.PublicKey().String()}); err == nil {
		t.Fatal("legacy deleted through managed controller")
	}
}
