package b02_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"darc-vpn/back-vpn/internal/accounts"
	"darc-vpn/back-vpn/internal/auth"
	profilecrypto "darc-vpn/back-vpn/internal/crypto"
	"darc-vpn/back-vpn/internal/domain"
	"darc-vpn/back-vpn/internal/profiles"
	"darc-vpn/back-vpn/internal/storage"
)

var testNow = time.Date(2026, 9, 29, 3, 0, 0, 0, time.UTC)

func TestAccountCreationCreatesOnePrimaryAndOnlyAdminCreatesExtras(t *testing.T) {
	ctx := context.Background()
	store := openStore(t, filepath.Join(t.TempDir(), "vpn.db"))
	services := newServices(t, store, bytes32(1))

	admin, err := services.accounts.BootstrapAdmin(ctx, "root", "correct horse")
	if err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}
	created, err := services.accounts.Create(ctx, admin, accounts.CreateRequest{
		Username: "friend-a", Password: "battery staple", PrimaryDisplayName: "Principal",
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	if created.ProfileCount != 1 || len(created.Profiles) != 1 || !created.Profiles[0].IsPrimary {
		t.Fatalf("new account must have exactly one primary profile: %#v", created)
	}
	if strings.Contains(mustJSON(t, created), "private") || strings.Contains(mustJSON(t, created), "preshared") {
		t.Fatalf("admin DTO leaked private material: %s", mustJSON(t, created))
	}
	var passwordHash string
	if err := store.DB().QueryRowContext(ctx, `SELECT password_hash FROM accounts WHERE id = ?`, created.ID).Scan(&passwordHash); err != nil {
		t.Fatalf("read password hash: %v", err)
	}
	if passwordHash == "battery staple" || !strings.HasPrefix(passwordHash, "$argon2id$") {
		t.Fatalf("password was not stored as Argon2id: %q", passwordHash)
	}

	friend := auth.Principal{AccountID: created.ID, Role: domain.RoleFriend}
	if _, err := services.accounts.CreateExtraProfile(ctx, friend, created.ID, "Phone"); domain.CodeOf(err) != domain.ErrForbidden {
		t.Fatalf("friend extra profile error = %v, want forbidden", err)
	}
	extra, err := services.accounts.CreateExtraProfile(ctx, admin, created.ID, "Phone")
	if err != nil {
		t.Fatalf("admin create extra profile: %v", err)
	}
	if extra.IsPrimary {
		t.Fatal("extra profile was marked primary")
	}
	if extra.PeerPublicKey == created.Profiles[0].PeerPublicKey {
		t.Fatal("slots reused a WireGuard key pair")
	}

	updated, err := services.accounts.Get(ctx, admin, created.ID)
	if err != nil {
		t.Fatalf("get account: %v", err)
	}
	if updated.ProfileCount != 2 || len(updated.Profiles) != 2 {
		t.Fatalf("profile count = %d/%d, want 2", updated.ProfileCount, len(updated.Profiles))
	}
}

func TestLoginStoresOnlyDigestAndRevalidatesBlockedAccount(t *testing.T) {
	ctx := context.Background()
	store := openStore(t, filepath.Join(t.TempDir(), "vpn.db"))
	services := newServices(t, store, bytes32(2))
	admin, err := services.accounts.BootstrapAdmin(ctx, "root", "admin-pass")
	if err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}
	created, err := services.accounts.Create(ctx, admin, accounts.CreateRequest{
		Username: "friend-a", Password: "friend-pass", PrimaryDisplayName: "Principal",
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}

	login, err := services.auth.Login(ctx, "friend-a", "friend-pass", domain.RoleFriend)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if login.AccessToken == "" || login.AccountID != created.ID || login.Role != domain.RoleFriend {
		t.Fatalf("unexpected login response: %#v", login)
	}
	digest := sha256.Sum256([]byte(login.AccessToken))
	var stored []byte
	if err := store.DB().QueryRowContext(ctx, `SELECT token_digest FROM sessions WHERE account_id = ?`, created.ID).Scan(&stored); err != nil {
		t.Fatalf("read session digest: %v", err)
	}
	if string(stored) == login.AccessToken || string(stored) != string(digest[:]) {
		t.Fatal("session token was not persisted exclusively as its SHA-256 digest")
	}

	principal, err := services.auth.Authenticate(ctx, login.AccessToken, domain.RoleFriend)
	if err != nil || principal.AccountID != created.ID {
		t.Fatalf("authenticate valid token: principal=%#v err=%v", principal, err)
	}
	if _, err := services.auth.Authenticate(ctx, login.AccessToken, domain.RoleAdmin); domain.CodeOf(err) != domain.ErrForbidden {
		t.Fatalf("role mismatch error = %v, want forbidden", err)
	}
	if _, err := services.auth.Authenticate(ctx, "missing-token", domain.RoleFriend); domain.CodeOf(err) != domain.ErrNotAuthenticated {
		t.Fatalf("missing token error = %v, want not_authenticated", err)
	}

	if err := store.SetAccountStatus(ctx, created.ID, domain.AccountBlocked); err != nil {
		t.Fatalf("block account: %v", err)
	}
	if _, err := services.auth.Authenticate(ctx, login.AccessToken, domain.RoleFriend); domain.CodeOf(err) != domain.ErrAccountBlocked {
		t.Fatalf("blocked session error = %v, want account_blocked", err)
	}

	for _, tc := range []struct{ username, password string }{{"friend-a", "wrong"}, {"unknown", "wrong"}} {
		_, err := services.auth.Login(ctx, tc.username, tc.password, domain.RoleFriend)
		if domain.CodeOf(err) != domain.ErrInvalidCredentials || err.Error() != "invalid_credentials: invalid credentials" {
			t.Fatalf("login %q exposed a distinct error: %v", tc.username, err)
		}
	}
}

func TestProfilesAreOwnerScopedEncryptedAndRecoverAfterRestart(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "vpn.db")
	masterKey := bytes32(3)
	store := openStore(t, dbPath)
	services := newServices(t, store, masterKey)
	admin, err := services.accounts.BootstrapAdmin(ctx, "root", "admin-pass")
	if err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}
	first, err := services.accounts.Create(ctx, admin, accounts.CreateRequest{Username: "one", Password: "pass-one", PrimaryDisplayName: "One"})
	if err != nil {
		t.Fatalf("create first account: %v", err)
	}
	second, err := services.accounts.Create(ctx, admin, accounts.CreateRequest{Username: "two", Password: "pass-two", PrimaryDisplayName: "Two"})
	if err != nil {
		t.Fatalf("create second account: %v", err)
	}

	var ciphertext []byte
	if err := store.DB().QueryRowContext(ctx, `SELECT private_ciphertext FROM profiles WHERE id = ?`, first.PrimaryID).Scan(&ciphertext); err != nil {
		t.Fatalf("read ciphertext: %v", err)
	}
	if len(ciphertext) == 0 || strings.Contains(string(ciphertext), "PrivateKey") || strings.Contains(string(ciphertext), "private") {
		t.Fatalf("private profile was not stored as opaque ciphertext: %q", ciphertext)
	}

	owner := auth.Principal{AccountID: first.ID, Role: domain.RoleFriend}
	privateProfile, err := services.profiles.Recover(ctx, owner, first.PrimaryID)
	if err != nil {
		t.Fatalf("recover own profile: %v", err)
	}
	if privateProfile.PrivateKey == "" || privateProfile.ID != first.PrimaryID {
		t.Fatalf("incomplete recovered profile: %#v", privateProfile)
	}
	if strings.Contains(string(ciphertext), privateProfile.PrivateKey) {
		t.Fatal("stored ciphertext contains the recovered WireGuard private key")
	}
	other := auth.Principal{AccountID: second.ID, Role: domain.RoleFriend}
	if _, err := services.profiles.Recover(ctx, other, first.PrimaryID); domain.CodeOf(err) != domain.ErrProfileNotFound {
		t.Fatalf("cross-account recovery error = %v, want profile_not_found", err)
	}
	if _, err := services.profiles.Recover(ctx, admin, first.PrimaryID); domain.CodeOf(err) != domain.ErrForbidden {
		t.Fatalf("admin private recovery error = %v, want forbidden", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("close first process store: %v", err)
	}
	reopened := openStore(t, dbPath)
	restarted := newServices(t, reopened, masterKey)
	login, err := restarted.auth.Login(ctx, "one", "pass-one", domain.RoleFriend)
	if err != nil {
		t.Fatalf("login after restart: %v", err)
	}
	principal, err := restarted.auth.Authenticate(ctx, login.AccessToken, domain.RoleFriend)
	if err != nil {
		t.Fatalf("authenticate after restart: %v", err)
	}
	recovered, err := restarted.profiles.Recover(ctx, principal, first.PrimaryID)
	if err != nil || recovered.PrivateKey != privateProfile.PrivateKey {
		t.Fatalf("recover after restart: profile=%#v err=%v", recovered, err)
	}
	if _, err := reopened.DB().ExecContext(ctx, `UPDATE profiles SET private_ciphertext = randomblob(length(private_ciphertext)) WHERE id = ?`, first.PrimaryID); err != nil {
		t.Fatalf("corrupt encrypted profile: %v", err)
	}
	if _, err := restarted.profiles.Recover(ctx, principal, first.PrimaryID); domain.CodeOf(err) != domain.ErrProfileRecoveryFailed {
		t.Fatalf("corrupt profile recovery error = %v, want profile_recovery_failed", err)
	}
}

func TestProfileCipherFailsClosedForWrongKeyCorruptionVersionAndMissingKey(t *testing.T) {
	ctx := context.Background()
	aad := profilecrypto.AssociatedData("account-1", "profile-1", profilecrypto.Version)
	cipherA, err := profilecrypto.NewAESGCM(bytes32(4), nil)
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	encrypted, err := cipherA.Encrypt(ctx, []byte(`{"privateKey":"secret"}`), aad)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	plain, err := cipherA.Decrypt(ctx, encrypted, aad)
	if err != nil || string(plain) != `{"privateKey":"secret"}` {
		t.Fatalf("round trip: plain=%q err=%v", plain, err)
	}
	second, err := cipherA.Encrypt(ctx, []byte(`{"privateKey":"secret"}`), aad)
	if err != nil {
		t.Fatalf("second encryption: %v", err)
	}
	if string(second.Nonce) == string(encrypted.Nonce) || string(second.Ciphertext) == string(encrypted.Ciphertext) {
		t.Fatal("AES-GCM nonce/ciphertext was reused across records")
	}

	cipherB, err := profilecrypto.NewAESGCM(bytes32(5), nil)
	if err != nil {
		t.Fatalf("new wrong-key cipher: %v", err)
	}
	assertRecoveryFailure(t, cipherB, encrypted, aad)
	corrupt := encrypted
	corrupt.Ciphertext = append([]byte(nil), encrypted.Ciphertext...)
	corrupt.Ciphertext[0] ^= 0xff
	assertRecoveryFailure(t, cipherA, corrupt, aad)
	unknown := encrypted
	unknown.Version = "unknown"
	assertRecoveryFailure(t, cipherA, unknown, aad)
	if _, err := profilecrypto.NewAESGCM(nil, nil); domain.CodeOf(err) != domain.ErrProfileRecoveryFailed {
		t.Fatalf("missing master key error = %v, want profile_recovery_failed", err)
	}
}

func TestProfileDeliveryFailsClosedWithoutAcceptanceGate(t *testing.T) {
	ctx := context.Background()
	store := openStore(t, filepath.Join(t.TempDir(), "gate.db"))
	services := newServices(t, store, bytes32(7))
	admin, err := services.accounts.BootstrapAdmin(ctx, "root", "admin-pass")
	if err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}
	account, err := services.accounts.Create(ctx, admin, accounts.CreateRequest{Username: "friend", Password: "pass", PrimaryDisplayName: "Principal"})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	cipher, err := profilecrypto.NewAESGCM(bytes32(7), nil)
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	closed := profiles.NewService(store, cipher, nil)
	_, err = closed.Recover(ctx, auth.Principal{AccountID: account.ID, Role: domain.RoleFriend}, account.PrimaryID)
	if domain.CodeOf(err) != domain.ErrTermsAcceptanceRequired {
		t.Fatalf("missing acceptance gate error = %v, want terms_acceptance_required", err)
	}
}

func TestOpaqueIDGeneratorProducesURLSafeNonSequentialIDs(t *testing.T) {
	generator := profilecrypto.NewOpaqueIDGenerator(nil)
	seen := make(map[string]bool)
	for index := 0; index < 32; index++ {
		value, err := generator.NewID()
		if err != nil {
			t.Fatalf("new opaque ID: %v", err)
		}
		if seen[value] || len(value) != 22 || strings.ContainsAny(value, "+/=") {
			t.Fatalf("invalid opaque ID %q", value)
		}
		seen[value] = true
	}
}

func TestFreshMigrationCreatesAllB02StateTables(t *testing.T) {
	store := openStore(t, filepath.Join(t.TempDir(), "fresh.db"))
	want := []string{"administrators", "accounts", "sessions", "profiles", "terms_acceptances", "revocation_operations", "revocation_peers"}
	for _, table := range want {
		var name string
		err := store.DB().QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name)
		if err != nil {
			t.Errorf("migration did not create %s: %v", table, err)
		}
	}
}

type serviceSet struct {
	auth     *auth.Service
	accounts *accounts.Service
	profiles *profiles.Service
}

func newServices(t *testing.T, store *storage.Store, masterKey []byte) serviceSet {
	t.Helper()
	hasher := auth.NewArgon2idHasher(auth.TestParams())
	cipher, err := profilecrypto.NewAESGCM(masterKey, nil)
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	factory := profiles.NewFactory(profiles.FactoryConfig{
		IDs:        &sequenceIDs{},
		Clock:      fixedClock{},
		Keys:       profiles.NewWireGuardKeyGenerator(nil),
		Cipher:     cipher,
		Address:    func(_, profileID string) string { return "10.8.0." + profileID[len(profileID)-1:] + "/32" },
		AllowedIPs: []string{"0.0.0.0/0"},
		DNS:        []string{"1.1.1.1"},
	})
	profileService := profiles.NewService(store, cipher, allowAcceptance{})
	accountService := accounts.NewService(store, hasher, factory, &sequenceIDs{}, fixedClock{})
	authService, err := auth.NewService(store, hasher, fixedClock{}, auth.Config{SessionTTL: time.Hour})
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}
	return serviceSet{auth: authService, accounts: accountService, profiles: profileService}
}

func openStore(t *testing.T, path string) *storage.Store {
	t.Helper()
	store, err := storage.Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

type sequenceIDs struct{ next int }

func (s *sequenceIDs) NewID() (string, error) {
	s.next++
	return "opaque-id-" + string(rune('0'+s.next)), nil
}

type fixedClock struct{}

func (fixedClock) Now() time.Time { return testNow }

type allowAcceptance struct{}

func (allowAcceptance) AuthorizeProfileDelivery(context.Context, string) error { return nil }

func bytes32(value byte) []byte {
	result := make([]byte, 32)
	for index := range result {
		result[index] = value
	}
	return result
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return strings.ToLower(string(encoded))
}

func assertRecoveryFailure(t *testing.T, cipher *profilecrypto.AESGCM, encrypted domain.EncryptedPrivateProfile, aad []byte) {
	t.Helper()
	_, err := cipher.Decrypt(context.Background(), encrypted, aad)
	if domain.CodeOf(err) != domain.ErrProfileRecoveryFailed {
		t.Fatalf("decrypt error = %v, want profile_recovery_failed", err)
	}
	if errors.Is(err, sql.ErrNoRows) {
		t.Fatal("cryptographic failure leaked an internal cause")
	}
}
