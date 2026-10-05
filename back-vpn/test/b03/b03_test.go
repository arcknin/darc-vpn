package b03_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"darc-vpn/back-vpn/internal/accounts"
	"darc-vpn/back-vpn/internal/audit"
	"darc-vpn/back-vpn/internal/auth"
	profilecrypto "darc-vpn/back-vpn/internal/crypto"
	"darc-vpn/back-vpn/internal/domain"
	httpapi "darc-vpn/back-vpn/internal/http"
	"darc-vpn/back-vpn/internal/profiles"
	"darc-vpn/back-vpn/internal/storage"
	"darc-vpn/back-vpn/internal/terms"
)

var now = time.Date(2026, 9, 29, 12, 34, 56, 789000000, time.UTC)

func TestAcceptanceReceiptIsExactAtomicRestrictedAndOpensGateOnlyAfterSuccess(t *testing.T) {
	ctx := context.Background()
	document := fixtureDocument(t)
	store := openStore(t)
	receiptDir := filepath.Join(t.TempDir(), "persistent", "acceptances")
	service := newTermsService(t, store, document, receiptDir, &sequenceIDs{prefix: "event"})
	principal := auth.Principal{AccountID: seedFriend(t, store), Role: domain.RoleFriend}

	if err := service.AuthorizeProfileDelivery(ctx, principal.AccountID); domain.CodeOf(err) != domain.ErrTermsAcceptanceRequired {
		t.Fatalf("gate before acceptance = %v, want terms_acceptance_required", err)
	}
	receipt, err := service.Accept(ctx, principal, terms.AcceptanceRequest{Accepted: true, Version: document.Version, SHA256: document.SHA256})
	if err != nil {
		t.Fatalf("accept terms: %v", err)
	}
	if err := service.AuthorizeProfileDelivery(ctx, principal.AccountID); err != nil {
		t.Fatalf("gate after persisted acceptance: %v", err)
	}

	path := filepath.Join(receiptDir, receipt.EventID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read receipt: %v", err)
	}
	var stored domain.TermsAcceptanceReceipt
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("receipt is not JSON: %v", err)
	}
	if !reflect.DeepEqual(stored, receipt) || stored.AccountID != principal.AccountID || stored.Version != document.Version || stored.SHA256 != document.SHA256 || !stored.Accepted || !stored.AcceptedAt.Equal(now) {
		t.Fatalf("receipt mismatch: got=%#v want=%#v", stored, receipt)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat receipt: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("receipt mode=%#o want 0600", info.Mode().Perm())
	}
	dirInfo, err := os.Stat(receiptDir)
	if err != nil {
		t.Fatalf("stat receipt dir: %v", err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("receipt dir mode=%#o want 0700", dirInfo.Mode().Perm())
	}
	entries, err := os.ReadDir(receiptDir)
	if err != nil {
		t.Fatalf("read receipt dir: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp-") {
			t.Fatalf("temporary receipt remained after commit: %s", entry.Name())
		}
	}
}

func TestAcceptanceFailsClosedForMismatchAndFileFailureAndCanRetry(t *testing.T) {
	ctx := context.Background()
	document := fixtureDocument(t)
	store := openStore(t)
	accountID := seedFriend(t, store)
	principal := auth.Principal{AccountID: accountID, Role: domain.RoleFriend}
	badDir := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(badDir, []byte("occupied"), 0o600); err != nil {
		t.Fatalf("prepare bad receipt path: %v", err)
	}
	service := newTermsService(t, store, document, badDir, &sequenceIDs{prefix: "event"})

	for _, input := range []terms.AcceptanceRequest{
		{Accepted: false, Version: document.Version, SHA256: document.SHA256},
		{Accepted: true, Version: "wrong", SHA256: document.SHA256},
		{Accepted: true, Version: document.Version, SHA256: strings.Repeat("0", 64)},
	} {
		if _, err := service.Accept(ctx, principal, input); domain.CodeOf(err) != domain.ErrInvalidRequest {
			t.Fatalf("mismatched acceptance error=%v want invalid_request", err)
		}
	}
	if _, err := service.Accept(ctx, principal, terms.AcceptanceRequest{Accepted: true, Version: document.Version, SHA256: document.SHA256}); domain.CodeOf(err) != domain.ErrAcceptancePersistence {
		t.Fatalf("file failure error=%v want acceptance_persistence_failed", err)
	}
	if err := service.AuthorizeProfileDelivery(ctx, accountID); domain.CodeOf(err) != domain.ErrTermsAcceptanceRequired {
		t.Fatalf("file failure opened gate: %v", err)
	}

	retry := newTermsService(t, store, document, filepath.Join(t.TempDir(), "receipts"), &sequenceIDs{prefix: "retry"})
	if _, err := retry.Accept(ctx, principal, terms.AcceptanceRequest{Accepted: true, Version: document.Version, SHA256: document.SHA256}); err != nil {
		t.Fatalf("retry acceptance: %v", err)
	}
	if err := retry.AuthorizeProfileDelivery(ctx, accountID); err != nil {
		t.Fatalf("retry did not open gate: %v", err)
	}
}

func TestDocumentConfigurationRejectsWrongHashAndEditorialDraft(t *testing.T) {
	contents := []byte("complete fictional terms")
	sum := sha256.Sum256(contents)
	if _, err := terms.NewDocument("v1", contents, hex.EncodeToString(sum[:])); err != nil {
		t.Fatalf("valid document: %v", err)
	}
	if _, err := terms.NewDocument("v1", contents, strings.Repeat("0", 64)); err == nil {
		t.Fatal("wrong configured hash was accepted")
	}
	draft := []byte("Responsável: [DEFINIR]\nContato: [E-MAIL]\n[PRAZO A DEFINIR]")
	draftSum := sha256.Sum256(draft)
	if _, err := terms.NewDocument("v1", draft, hex.EncodeToString(draftSum[:])); err == nil {
		t.Fatal("editorially incomplete draft was accepted")
	}
	invalidUTF8 := []byte{0xff, 0xfe}
	invalidSum := sha256.Sum256(invalidUTF8)
	if _, err := terms.NewDocument("v1", invalidUTF8, hex.EncodeToString(invalidSum[:])); err == nil {
		t.Fatal("non-UTF-8 terms were accepted even though JSON could not present the exact bytes")
	}
}

func TestHTTPGateAuditAndTrustedProxyRules(t *testing.T) {
	ctx := context.Background()
	document := fixtureDocument(t)
	store := openStore(t)
	services := buildServices(t, store, document, filepath.Join(t.TempDir(), "receipts"))
	admin, err := services.accounts.BootstrapAdmin(ctx, "root", "admin-pass")
	if err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}
	created, err := services.accounts.Create(ctx, admin, accounts.CreateRequest{Username: "friend", Password: "friend-pass", PrimaryDisplayName: "Principal"})
	if err != nil {
		t.Fatalf("create friend: %v", err)
	}
	var logs bytes.Buffer
	sink := audit.NewJSONSink(&logs)
	recorder := audit.NewRecorder(sink, fixedClock{}, &sequenceIDs{prefix: "audit"})
	trusted, _ := netip.ParsePrefix("10.0.0.0/8")
	handler := httpapi.New(services.auth, services.accounts, services.profiles,
		httpapi.WithTerms(services.terms),
		httpapi.WithAudit(recorder, audit.NewSourceIPResolver([]netip.Prefix{trusted})),
	)

	friendToken := login(t, handler, "/api/v1/auth/friend/login", "friend", "friend-pass", "203.0.113.9:4231", "198.51.100.4")
	before := request(t, handler, http.MethodGet, "/api/v1/profiles", friendToken, nil, "203.0.113.9:4231", "198.51.100.4")
	if before.Code != http.StatusPreconditionRequired || !strings.Contains(before.Body.String(), string(domain.ErrTermsAcceptanceRequired)) {
		t.Fatalf("profiles before acceptance status=%d body=%s", before.Code, before.Body.String())
	}
	current := request(t, handler, http.MethodGet, "/api/v1/terms/current", friendToken, nil, "10.1.2.3:8123", "198.51.100.4")
	if current.Code != http.StatusOK || !bytes.Equal(extractText(t, current.Body.Bytes()), []byte(document.Text)) {
		t.Fatalf("current terms status=%d body=%s", current.Code, current.Body.String())
	}
	accepted := request(t, handler, http.MethodPost, "/api/v1/terms/acceptance", friendToken, terms.AcceptanceRequest{Accepted: true, Version: document.Version, SHA256: document.SHA256}, "203.0.113.9:4231", "198.51.100.4")
	if accepted.Code != http.StatusCreated {
		t.Fatalf("acceptance status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	private := request(t, handler, http.MethodGet, "/api/v1/profiles/"+created.PrimaryID+"/private", friendToken, nil, "203.0.113.9:4231", "198.51.100.4")
	if private.Code != http.StatusOK || !strings.Contains(private.Body.String(), "privateKey") {
		t.Fatalf("private profile after acceptance status=%d body=%s", private.Code, private.Body.String())
	}

	lines := strings.Split(strings.TrimSpace(logs.String()), "\n")
	if len(lines) < 5 {
		t.Fatalf("missing audit records: %s", logs.String())
	}
	seenRemote, seenTrustedProxy, seenProfile := false, false, false
	for _, line := range lines {
		var metadata map[string]any
		if err := json.Unmarshal([]byte(line), &metadata); err != nil {
			t.Fatalf("invalid JSON audit line %q: %v", line, err)
		}
		for key := range metadata {
			switch key {
			case "event_id", "event_type", "account_id", "profile_id", "source_ip", "timestamp_utc":
			default:
				t.Fatalf("unapproved audit field %q in %s", key, line)
			}
		}
		seenRemote = seenRemote || metadata["source_ip"] == "203.0.113.9"
		seenTrustedProxy = seenTrustedProxy || metadata["source_ip"] == "198.51.100.4"
		seenProfile = seenProfile || metadata["profile_id"] == created.PrimaryID
	}
	if !seenRemote || !seenTrustedProxy || !seenProfile {
		t.Fatalf("audit attribution missing remote=%v trusted-proxy=%v profile=%v logs=%s", seenRemote, seenTrustedProxy, seenProfile, logs.String())
	}
	for _, forbidden := range []string{"privatekey", "presharedkey", "authorization", "friend-pass", "vpn.darcanix.com", "complete fictional", "payload", "destination", "host"} {
		if strings.Contains(strings.ToLower(logs.String()), forbidden) {
			t.Fatalf("audit logs leaked forbidden %q: %s", forbidden, logs.String())
		}
	}
}

func TestVPNEventsRequireControllerAuthenticationAndStayMinimal(t *testing.T) {
	var logs bytes.Buffer
	recorder := audit.NewRecorder(audit.NewJSONSink(&logs), fixedClock{}, &sequenceIDs{prefix: "vpn"})
	controller, err := audit.NewVPNEventPort("controller-secret", recorder)
	if err != nil {
		t.Fatalf("new VPN event port: %v", err)
	}
	if err := controller.Record(context.Background(), "wrong", "account-1", "profile-1", "192.0.2.10"); !errors.Is(err, audit.ErrControllerUnauthenticated) {
		t.Fatalf("unauthenticated controller error=%v", err)
	}
	if logs.Len() != 0 {
		t.Fatalf("unauthenticated event was logged: %s", logs.String())
	}
	if err := controller.Record(context.Background(), "controller-secret", "account-1", "profile-1", "192.0.2.10"); err != nil {
		t.Fatalf("authenticated VPN event: %v", err)
	}
	if strings.Contains(logs.String(), "controller-secret") || !strings.Contains(logs.String(), `"event_type":"vpn_event"`) {
		t.Fatalf("unsafe VPN audit record: %s", logs.String())
	}
}

type services struct {
	auth     *auth.Service
	accounts *accounts.Service
	profiles *profiles.Service
	terms    *terms.Service
}

func buildServices(t *testing.T, store *storage.Store, document domain.TermsDocument, receiptDir string) services {
	t.Helper()
	ids := &sequenceIDs{prefix: "id"}
	clock := fixedClock{}
	cipher, err := profilecrypto.NewAESGCM(bytes.Repeat([]byte{7}, 32), nil)
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	factory := profiles.NewFactory(profiles.FactoryConfig{IDs: ids, Clock: clock, Keys: profiles.NewWireGuardKeyGenerator(nil), Cipher: cipher, Address: func(_, _ string) string { return "10.8.0.2/32" }, AllowedIPs: []string{"0.0.0.0/0"}, DNS: []string{"1.1.1.1"}})
	accountService := accounts.NewService(store, auth.NewArgon2idHasher(auth.TestParams()), factory, ids, clock)
	authService, err := auth.NewService(store, auth.NewArgon2idHasher(auth.TestParams()), clock, auth.Config{SessionTTL: time.Hour})
	if err != nil {
		t.Fatalf("auth service: %v", err)
	}
	termsService := newTermsService(t, store, document, receiptDir, ids)
	return services{auth: authService, accounts: accountService, profiles: profiles.NewService(store, cipher, termsService), terms: termsService}
}

func newTermsService(t *testing.T, store *storage.Store, document domain.TermsDocument, dir string, ids domain.IDGenerator) *terms.Service {
	t.Helper()
	receipts, err := terms.NewFileReceiptStore(dir)
	if err != nil {
		t.Fatalf("receipt store: %v", err)
	}
	service, err := terms.NewService(document, store, receipts, fixedClock{}, ids)
	if err != nil {
		t.Fatalf("terms service: %v", err)
	}
	return service
}

func fixtureDocument(t *testing.T) domain.TermsDocument {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "terms", "complete-fictional-v1.txt"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	sum := sha256.Sum256(data)
	document, err := terms.NewDocument("test-v1", data, hex.EncodeToString(sum[:]))
	if err != nil {
		t.Fatalf("fixture document: %v", err)
	}
	return document
}

func openStore(t *testing.T) *storage.Store {
	t.Helper()
	store, err := storage.Open(filepath.Join(t.TempDir(), "vpn.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func seedFriend(t *testing.T, store *storage.Store) string {
	t.Helper()
	svc := buildServices(t, store, fixtureDocument(t), filepath.Join(t.TempDir(), "seed-receipts"))
	admin, err := svc.accounts.BootstrapAdmin(context.Background(), "seed-admin", "pass")
	if err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	created, err := svc.accounts.Create(context.Background(), admin, accounts.CreateRequest{Username: "seed-friend", Password: "pass", PrimaryDisplayName: "Primary"})
	if err != nil {
		t.Fatalf("seed friend: %v", err)
	}
	return created.ID
}

type fixedClock struct{}

func (fixedClock) Now() time.Time { return now }

type sequenceIDs struct {
	prefix string
	next   int
}

func (s *sequenceIDs) NewID() (string, error) {
	s.next++
	return s.prefix + "-" + string(rune('a'+s.next-1)), nil
}

func login(t *testing.T, handler http.Handler, path, username, password, remote, forwarded string) string {
	t.Helper()
	response := request(t, handler, http.MethodPost, path, "", map[string]string{"username": username, "password": password}, remote, forwarded)
	if response.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", response.Code, response.Body.String())
	}
	var result struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	return result.AccessToken
}

func request(t *testing.T, handler http.Handler, method, path, token string, body any, remote, forwarded string) *httptest.ResponseRecorder {
	t.Helper()
	var input bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&input).Encode(body); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &input)
	req.RemoteAddr = remote
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if forwarded != "" {
		req.Header.Set("X-Forwarded-For", forwarded)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}

func extractText(t *testing.T, data []byte) []byte {
	t.Helper()
	var document domain.TermsDocument
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("decode terms: %v", err)
	}
	return []byte(document.Text)
}
