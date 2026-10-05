package storage

import (
	"context"
	"darc-vpn/back-vpn/internal/domain"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func managedStore(t *testing.T, pool string) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if err = s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err = s.ConfigureManaged(context.Background(), pool, "fd42:77:77::/64"); err != nil {
		t.Fatal(err)
	}
	return s
}
func createManaged(s *Store, n int) error {
	id := fmt.Sprint(n)
	return s.CreateAccountWithPrimary(context.Background(), domain.AccountRecord{ID: id, Username: id, Status: domain.AccountActive, PrimaryID: id, CreatedAt: time.Now()}, "hash", domain.ProfileRecord{ID: id, AccountID: id, DisplayName: id, IsPrimary: true, Endpoint: domain.VPNEndpoint, PeerPublicKey: "key" + id, Address: "legacy", EncryptedPrivate: domain.EncryptedPrivateProfile{Version: "v1", Nonce: []byte{1}, Ciphertext: []byte{2}}, CreatedAt: time.Now()})
}
func TestManagedAllocationIsAtomicUniqueAndNeverReusesRevoked(t *testing.T) {
	s := managedStore(t, "10.77.77.0/29")
	var wg sync.WaitGroup
	errs := make(chan error, 5)
	for n := 0; n < 5; n++ {
		wg.Add(1)
		go func(n int) { defer wg.Done(); errs <- createManaged(s, n) }(n)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	peers, err := s.ManagedPeers(context.Background())
	if err != nil || len(peers) != 5 {
		t.Fatalf("peers=%v err=%v", len(peers), err)
	}
	seen := map[string]bool{}
	for _, p := range peers {
		if p.State != "pending" || seen[p.Address] {
			t.Fatal("nonunique/nonpending")
		}
		seen[p.Address] = true
	}
	if _, _, err = s.BeginAccountBlock(context.Background(), "0", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err = createManaged(s, 10); domain.CodeOf(err) != domain.ErrPoolExhausted {
		t.Fatalf("want pool exhaustion, got %v", err)
	}
	if _, err = s.Account(context.Background(), "10"); err == nil {
		t.Fatal("partial account persisted")
	}
	peers, _ = s.ManagedPeers(context.Background())
	for _, p := range peers {
		if p.AccountID == "0" && p.State != "revoked" {
			t.Fatal("blocked desired peer not revoked")
		}
	}
}
func TestManagedMigrationRejectsIncompatibleLegacyWithoutRewrite(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err = createManaged(s, 0); err != nil {
		t.Fatal(err)
	}
	if err = s.ConfigureManaged(context.Background(), "10.77.77.0/24", "fd42:77:77::/64"); err == nil {
		t.Fatal("legacy malformed address accepted")
	}
	p, _ := s.Profile(context.Background(), "0", "0")
	if p.Address != "legacy" {
		t.Fatal("silently rewrote legacy profile")
	}
}
