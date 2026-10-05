package b04_test

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"darc-vpn/back-vpn/internal/accounts"
	"darc-vpn/back-vpn/internal/auth"
	profilecrypto "darc-vpn/back-vpn/internal/crypto"
	"darc-vpn/back-vpn/internal/domain"
	"darc-vpn/back-vpn/internal/profiles"
	"darc-vpn/back-vpn/internal/revocation"
	"darc-vpn/back-vpn/internal/storage"
	"darc-vpn/back-vpn/internal/wireguard"
)

var testNow = time.Date(2026, 9, 29, 15, 0, 0, 0, time.UTC)

func TestBlockCommitsAccessCutAndCompleteSnapshotBeforeController(t *testing.T) {
	fixture := newFixture(t)
	controller := newBlockingController()
	service := revocation.NewService(fixture.store, controller, fixedClock{})

	result := make(chan blockResult, 1)
	go func() {
		operation, err := service.Block(context.Background(), fixture.admin, fixture.account.ID)
		result <- blockResult{operation: operation, err: err}
	}()

	peerIDs := <-controller.entered
	wantPeerIDs := accountPeerIDs(fixture.account)
	if !reflect.DeepEqual(sorted(peerIDs), sorted(wantPeerIDs)) {
		t.Fatalf("controller peers=%v want=%v", peerIDs, wantPeerIDs)
	}

	var status string
	if err := fixture.store.DB().QueryRow(`SELECT status FROM accounts WHERE id = ?`, fixture.account.ID).Scan(&status); err != nil {
		t.Fatalf("read account status while controller waits: %v", err)
	}
	if status != string(domain.AccountBlocked) {
		t.Fatalf("account status=%q before controller returns, want blocked", status)
	}
	var sessions int
	if err := fixture.store.DB().QueryRow(`SELECT COUNT(*) FROM sessions WHERE account_id = ?`, fixture.account.ID).Scan(&sessions); err != nil {
		t.Fatalf("count sessions while controller waits: %v", err)
	}
	if sessions != 0 {
		t.Fatalf("sessions remain usable before controller returns: %d", sessions)
	}
	if _, err := fixture.auth.Authenticate(context.Background(), fixture.friendToken, domain.RoleFriend); domain.CodeOf(err) != domain.ErrNotAuthenticated {
		t.Fatalf("session after committed block error=%v want not_authenticated", err)
	}
	pending, err := fixture.store.Revocation(context.Background(), fixture.account.ID)
	if err != nil {
		t.Fatalf("read durable pending operation: %v", err)
	}
	assertOperation(t, pending, domain.RevocationPending, len(wantPeerIDs))

	close(controller.release)
	completed := <-result
	if completed.err != nil {
		t.Fatalf("block account: %v", completed.err)
	}
	assertOperation(t, completed.operation, domain.RevocationCompleted, len(wantPeerIDs))
	stored, err := fixture.store.Revocation(context.Background(), fixture.account.ID)
	if err != nil {
		t.Fatalf("read completed operation: %v", err)
	}
	if !reflect.DeepEqual(stored, completed.operation) {
		t.Fatalf("persisted operation=%#v returned=%#v", stored, completed.operation)
	}
}

func TestCompletionRequiresActiveAndPersistentConfirmationForEveryPeer(t *testing.T) {
	tests := []struct {
		name string
		mode wireguard.FakeMode
		want domain.RevocationStatus
	}{
		{name: "confirmed", mode: wireguard.FakeConfirmAll, want: domain.RevocationCompleted},
		{name: "active removal without persistent confirmation", mode: wireguard.FakeOmitConfirmation, want: domain.RevocationPending},
		{name: "partial active removal", mode: wireguard.FakePartial, want: domain.RevocationPending},
		{name: "controller error", mode: wireguard.FakeFailure, want: domain.RevocationFailed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newFixture(t)
			controller := wireguard.NewFake(test.mode)
			service := revocation.NewService(fixture.store, controller, fixedClock{})

			operation, err := service.Block(context.Background(), fixture.admin, fixture.account.ID)
			if err != nil {
				t.Fatalf("block account: %v", err)
			}
			assertOperation(t, operation, test.want, len(fixture.account.Profiles))
			stored, err := service.Get(context.Background(), fixture.admin, fixture.account.ID)
			if err != nil {
				t.Fatalf("get revocation: %v", err)
			}
			if stored.Status != test.want {
				t.Fatalf("stored status=%q want=%q", stored.Status, test.want)
			}
			if test.want != domain.RevocationCompleted && stored.Error == "" {
				t.Fatal("non-completed operation omitted its sanitized reason")
			}
			encoded, err := json.Marshal(stored)
			if err != nil {
				t.Fatalf("marshal operation: %v", err)
			}
			lower := strings.ToLower(string(encoded))
			for _, forbidden := range []string{"privatekey", "presharedkey", "ciphertext", "simulated wireguard"} {
				if strings.Contains(lower, forbidden) {
					t.Fatalf("revocation response leaked %q: %s", forbidden, encoded)
				}
			}
		})
	}
}

func TestConcurrentAndRepeatedBlockUsesOneDurableOperation(t *testing.T) {
	fixture := newFixture(t)
	controller := wireguard.NewFake(wireguard.FakeConfirmAll)
	service := revocation.NewService(fixture.store, controller, fixedClock{})
	start := make(chan struct{})
	results := make(chan blockResult, 2)
	var wait sync.WaitGroup
	for index := 0; index < 2; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			operation, err := service.Block(context.Background(), fixture.admin, fixture.account.ID)
			results <- blockResult{operation: operation, err: err}
		}()
	}
	close(start)
	wait.Wait()
	close(results)
	for result := range results {
		if result.err != nil {
			t.Fatalf("concurrent block: %v", result.err)
		}
		assertOperation(t, result.operation, domain.RevocationCompleted, len(fixture.account.Profiles))
	}
	if calls := controller.RemoveCallCount(); calls != 1 {
		t.Fatalf("controller removal calls=%d want=1", calls)
	}
	var operations, peers int
	if err := fixture.store.DB().QueryRow(`SELECT COUNT(*) FROM revocation_operations WHERE account_id = ?`, fixture.account.ID).Scan(&operations); err != nil {
		t.Fatalf("count operations: %v", err)
	}
	if err := fixture.store.DB().QueryRow(`SELECT COUNT(*) FROM revocation_peers WHERE account_id = ?`, fixture.account.ID).Scan(&peers); err != nil {
		t.Fatalf("count peers: %v", err)
	}
	if operations != 1 || peers != len(fixture.account.Profiles) {
		t.Fatalf("durable rows operations=%d peers=%d", operations, peers)
	}
	if _, err := fixture.auth.Authenticate(context.Background(), fixture.friendToken, domain.RoleFriend); domain.CodeOf(err) != domain.ErrNotAuthenticated {
		t.Fatalf("session survived concurrent block: %v", err)
	}
}

func TestControllerFailureIsSanitized(t *testing.T) {
	fixture := newFixture(t)
	service := revocation.NewService(fixture.store, leakingController{}, fixedClock{})
	operation, err := service.Block(context.Background(), fixture.admin, fixture.account.ID)
	if err != nil {
		t.Fatalf("block account: %v", err)
	}
	if operation.Status != domain.RevocationFailed || operation.Error == "" {
		t.Fatalf("failure operation=%#v", operation)
	}
	if strings.Contains(operation.Error, "sensitive-adapter-detail") {
		t.Fatalf("adapter error leaked through operation: %#v", operation)
	}
}

type fixture struct {
	store       *storage.Store
	auth        *auth.Service
	admin       auth.Principal
	account     domain.AdminAccountMetadata
	friendToken string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	store, err := storage.Open(filepath.Join(t.TempDir(), "vpn.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ids := &sequenceIDs{}
	cipher, err := profilecrypto.NewAESGCM(bytes.Repeat([]byte{4}, 32), nil)
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	factory := profiles.NewFactory(profiles.FactoryConfig{
		IDs: ids, Clock: fixedClock{}, Keys: profiles.NewWireGuardKeyGenerator(nil), Cipher: cipher,
		Address:    func(_, profileID string) string { return "10.8.0." + profileID[len(profileID)-1:] + "/32" },
		AllowedIPs: []string{"0.0.0.0/0"}, DNS: []string{"1.1.1.1"},
	})
	hasher := auth.NewArgon2idHasher(auth.TestParams())
	accountService := accounts.NewService(store, hasher, factory, ids, fixedClock{})
	authService, err := auth.NewService(store, hasher, fixedClock{}, auth.Config{SessionTTL: time.Hour})
	if err != nil {
		t.Fatalf("auth service: %v", err)
	}
	admin, err := accountService.BootstrapAdmin(context.Background(), "root", "admin-pass")
	if err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}
	account, err := accountService.Create(context.Background(), admin, accounts.CreateRequest{Username: "friend", Password: "friend-pass", PrimaryDisplayName: "Primary"})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	extra, err := accountService.CreateExtraProfile(context.Background(), admin, account.ID, "Extra")
	if err != nil {
		t.Fatalf("create extra profile: %v", err)
	}
	account.Profiles = append(account.Profiles, extra)
	account.ProfileCount = len(account.Profiles)
	login, err := authService.Login(context.Background(), "friend", "friend-pass", domain.RoleFriend)
	if err != nil {
		t.Fatalf("friend login: %v", err)
	}
	return fixture{store: store, auth: authService, admin: admin, account: account, friendToken: login.AccessToken}
}

type sequenceIDs struct {
	mu   sync.Mutex
	next int
}

func (s *sequenceIDs) NewID() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	return "opaque-id-" + string(rune('a'+s.next-1)), nil
}

type fixedClock struct{}

func (fixedClock) Now() time.Time { return testNow }

type blockResult struct {
	operation domain.RevocationOperation
	err       error
}

type blockingController struct {
	entered chan []string
	release chan struct{}
}

func newBlockingController() *blockingController {
	return &blockingController{entered: make(chan []string, 1), release: make(chan struct{})}
}

func (c *blockingController) RemovePeers(ctx context.Context, peerIDs []string) ([]domain.WireGuardRemoval, error) {
	c.entered <- append([]string(nil), peerIDs...)
	select {
	case <-c.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	result := make([]domain.WireGuardRemoval, 0, len(peerIDs))
	for _, peerID := range peerIDs {
		result = append(result, domain.WireGuardRemoval{PeerID: peerID, Removed: true})
	}
	return result, nil
}

func (*blockingController) ConfirmPeersRemoved(context.Context, []string) (bool, error) {
	return true, nil
}

type leakingController struct{}

func (leakingController) RemovePeers(_ context.Context, peerIDs []string) ([]domain.WireGuardRemoval, error) {
	result := make([]domain.WireGuardRemoval, 0, len(peerIDs))
	for index, peerID := range peerIDs {
		removal := domain.WireGuardRemoval{PeerID: peerID, Removed: true}
		if index == 0 {
			removal.Removed = false
			removal.Error = "sensitive-adapter-detail"
		}
		result = append(result, removal)
	}
	return result, nil
}

func (leakingController) ConfirmPeersRemoved(context.Context, []string) (bool, error) {
	panic("persistent confirmation must not run after an active removal error")
}

func accountPeerIDs(account domain.AdminAccountMetadata) []string {
	result := make([]string, 0, len(account.Profiles))
	for _, profile := range account.Profiles {
		result = append(result, profile.PeerPublicKey)
	}
	return result
}

func sorted(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

func assertOperation(t *testing.T, operation domain.RevocationOperation, status domain.RevocationStatus, peerCount int) {
	t.Helper()
	if operation.Status != status || len(operation.Peers) != peerCount || operation.AccountID == "" || operation.UpdatedAt.IsZero() {
		t.Fatalf("operation=%#v want status=%q peers=%d", operation, status, peerCount)
	}
	if status == domain.RevocationCompleted {
		if operation.Error != "" {
			t.Fatalf("completed operation has error: %#v", operation)
		}
		for _, peer := range operation.Peers {
			if !peer.Removed || peer.Error != "" {
				t.Fatalf("completed operation has unconfirmed peer: %#v", operation)
			}
		}
	}
}
