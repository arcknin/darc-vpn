package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"darc-vpn/back-vpn/internal/accounts"
	"darc-vpn/back-vpn/internal/auth"
	"darc-vpn/back-vpn/internal/domain"
	"darc-vpn/back-vpn/internal/terms"
	"darc-vpn/back-vpn/internal/wireguard"
)

func TestConfiguredCommandServesB02AndB03RoutesOnLoopback(t *testing.T) {
	configuration := testConfig(t)
	var auditLog bytes.Buffer
	app, err := newApplication(context.Background(), configuration, &auditLog)
	if err != nil {
		t.Fatalf("compose application: %v", err)
	}
	t.Cleanup(func() { _ = app.Close() })
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on loopback: %v", err)
	}
	server := &http.Server{Handler: app.handler}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		<-done
	})
	baseURL := "http://" + listener.Addr().String()

	health := call(t, http.MethodGet, baseURL+"/healthz", "", nil)
	assertStatusNotReserved(t, health, http.StatusOK)
	adminToken := loginOverHTTP(t, baseURL+"/api/v1/auth/admin/login", "root", "admin-pass")
	created := call(t, http.MethodPost, baseURL+"/api/v1/admin/accounts", adminToken, accounts.CreateRequest{Username: "friend", Password: "friend-pass", PrimaryDisplayName: "Principal"})
	assertStatusNotReserved(t, created, http.StatusCreated)
	var account domain.AdminAccountMetadata
	decodeResponse(t, created, &account)
	friendToken := loginOverHTTP(t, baseURL+"/api/v1/auth/friend/login", "friend", "friend-pass")

	before := call(t, http.MethodGet, baseURL+"/api/v1/profiles", friendToken, nil)
	assertStatusNotReserved(t, before, http.StatusPreconditionRequired)
	current := call(t, http.MethodGet, baseURL+"/api/v1/terms/current", friendToken, nil)
	assertStatusNotReserved(t, current, http.StatusOK)
	var document domain.TermsDocument
	decodeResponse(t, current, &document)
	accepted := call(t, http.MethodPost, baseURL+"/api/v1/terms/acceptance", friendToken, terms.AcceptanceRequest{Accepted: true, Version: document.Version, SHA256: document.SHA256})
	assertStatusNotReserved(t, accepted, http.StatusCreated)
	profilesResponse := call(t, http.MethodGet, baseURL+"/api/v1/profiles", friendToken, nil)
	assertStatusNotReserved(t, profilesResponse, http.StatusOK)
	private := call(t, http.MethodGet, baseURL+"/api/v1/profiles/"+account.PrimaryID+"/private", friendToken, nil)
	assertStatusNotReserved(t, private, http.StatusOK)
	if !strings.Contains(readAndClose(t, private), "privateKey") {
		t.Fatal("private recovery route did not return the owned configuration")
	}
	if !strings.Contains(auditLog.String(), `"event_type":"authentication"`) || !strings.Contains(auditLog.String(), `"event_type":"authenticated_call"`) {
		t.Fatalf("composition did not emit required metadata: %s", auditLog.String())
	}
}

func TestConfiguredCommandServesB04RevocationStatesOnLoopback(t *testing.T) {
	tests := []struct {
		name string
		mode wireguard.FakeMode
		want domain.RevocationStatus
	}{
		{name: "pending", mode: wireguard.FakePartial, want: domain.RevocationPending},
		{name: "failed", mode: wireguard.FakeFailure, want: domain.RevocationFailed},
		{name: "completed", mode: wireguard.FakeConfirmAll, want: domain.RevocationCompleted},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configuration := testConfig(t)
			configuration.wireGuardController = wireguard.NewFake(test.mode)
			app, err := newApplication(context.Background(), configuration, io.Discard)
			if err != nil {
				t.Fatalf("compose application: %v", err)
			}
			t.Cleanup(func() { _ = app.Close() })
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatalf("listen on loopback: %v", err)
			}
			server := &http.Server{Handler: app.handler}
			done := make(chan error, 1)
			go func() { done <- server.Serve(listener) }()
			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				_ = server.Shutdown(ctx)
				<-done
			})
			baseURL := "http://" + listener.Addr().String()

			adminToken := loginOverHTTP(t, baseURL+"/api/v1/auth/admin/login", "root", "admin-pass")
			created := call(t, http.MethodPost, baseURL+"/api/v1/admin/accounts", adminToken, accounts.CreateRequest{Username: "friend", Password: "friend-pass", PrimaryDisplayName: "Principal"})
			assertStatusNotReserved(t, created, http.StatusCreated)
			var account domain.AdminAccountMetadata
			decodeResponse(t, created, &account)
			extra := call(t, http.MethodPost, baseURL+"/api/v1/admin/accounts/"+account.ID+"/profiles", adminToken, map[string]string{"displayName": "Extra"})
			assertStatusNotReserved(t, extra, http.StatusCreated)
			friendToken := loginOverHTTP(t, baseURL+"/api/v1/auth/friend/login", "friend", "friend-pass")

			blocked := call(t, http.MethodPost, baseURL+"/api/v1/admin/accounts/"+account.ID+"/block", adminToken, nil)
			assertStatusNotReserved(t, blocked, http.StatusAccepted)
			var operation domain.RevocationOperation
			decodeResponse(t, blocked, &operation)
			if operation.Status != test.want || len(operation.Peers) != 2 {
				t.Fatalf("block operation=%#v want status=%q with two peers", operation, test.want)
			}

			denied := call(t, http.MethodGet, baseURL+"/api/v1/profiles", friendToken, nil)
			if denied.StatusCode != http.StatusUnauthorized && denied.StatusCode != http.StatusLocked {
				t.Fatalf("friend session remained usable status=%d body=%s", denied.StatusCode, readAndClose(t, denied))
			}
			_ = denied.Body.Close()
			queried := call(t, http.MethodGet, baseURL+"/api/v1/admin/accounts/"+account.ID+"/revocation", adminToken, nil)
			assertStatusNotReserved(t, queried, http.StatusOK)
			var persisted domain.RevocationOperation
			decodeResponse(t, queried, &persisted)
			if persisted.Status != test.want || len(persisted.Peers) != 2 {
				t.Fatalf("queried operation=%#v want status=%q with two peers", persisted, test.want)
			}
		})
	}
}

func TestCommandFailsClosedForIncompleteOrDraftTermsConfiguration(t *testing.T) {
	configuration := testConfig(t)
	configuration.termsSHA256 = strings.Repeat("0", 64)
	if _, err := newApplication(context.Background(), configuration, io.Discard); err == nil {
		t.Fatal("application started with mismatched terms hash")
	}
	draft := []byte("Versão: [DEFINIR]\nContato: [E-MAIL]\n[PRAZO A DEFINIR]")
	draftPath := filepath.Join(t.TempDir(), "draft.md")
	if err := os.WriteFile(draftPath, draft, 0o600); err != nil {
		t.Fatalf("write draft: %v", err)
	}
	digest := sha256.Sum256(draft)
	configuration.termsPath = draftPath
	configuration.termsSHA256 = hex.EncodeToString(digest[:])
	if _, err := newApplication(context.Background(), configuration, io.Discard); err == nil {
		t.Fatal("application started with editorially incomplete terms")
	}
}

func TestConfiguredCommandReopensPersistedAcceptance(t *testing.T) {
	configuration := testConfig(t)
	first, err := newApplication(context.Background(), configuration, io.Discard)
	if err != nil {
		t.Fatalf("first application start: %v", err)
	}
	adminToken := loginOnHandler(t, first.handler, "/api/v1/auth/admin/login", "root", "admin-pass")
	created := serve(t, first.handler, http.MethodPost, "/api/v1/admin/accounts", adminToken, accounts.CreateRequest{Username: "friend", Password: "friend-pass", PrimaryDisplayName: "Principal"})
	if created.Code != http.StatusCreated {
		t.Fatalf("create account status=%d body=%s", created.Code, created.Body.String())
	}
	friendToken := loginOnHandler(t, first.handler, "/api/v1/auth/friend/login", "friend", "friend-pass")
	current := serve(t, first.handler, http.MethodGet, "/api/v1/terms/current", friendToken, nil)
	var document domain.TermsDocument
	if err := json.Unmarshal(current.Body.Bytes(), &document); err != nil {
		t.Fatalf("decode current terms: %v", err)
	}
	accepted := serve(t, first.handler, http.MethodPost, "/api/v1/terms/acceptance", friendToken, terms.AcceptanceRequest{Accepted: true, Version: document.Version, SHA256: document.SHA256})
	if accepted.Code != http.StatusCreated {
		t.Fatalf("accept status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close first application: %v", err)
	}

	second, err := newApplication(context.Background(), configuration, io.Discard)
	if err != nil {
		t.Fatalf("restart application: %v", err)
	}
	t.Cleanup(func() { _ = second.Close() })
	friendToken = loginOnHandler(t, second.handler, "/api/v1/auth/friend/login", "friend", "friend-pass")
	profiles := serve(t, second.handler, http.MethodGet, "/api/v1/profiles", friendToken, nil)
	if profiles.Code != http.StatusOK {
		t.Fatalf("persisted acceptance did not reopen gate status=%d body=%s", profiles.Code, profiles.Body.String())
	}
}

func testConfig(t *testing.T) config {
	t.Helper()
	termsPath, err := filepath.Abs(filepath.Join("..", "..", "testdata", "terms", "complete-fictional-v1.txt"))
	if err != nil {
		t.Fatalf("absolute fixture path: %v", err)
	}
	contents, err := os.ReadFile(termsPath)
	if err != nil {
		t.Fatalf("read terms fixture: %v", err)
	}
	digest := sha256.Sum256(contents)
	directory := t.TempDir()
	return config{
		wireGuardMode: "fake", mode: "local", address: "127.0.0.1:0", databasePath: filepath.Join(directory, "vpn.db"), masterKey: bytes.Repeat([]byte{9}, 32),
		termsVersion: "test-v1", termsPath: termsPath, termsSHA256: hex.EncodeToString(digest[:]),
		receiptDirectory: filepath.Join(directory, "persistent", "acceptances"), adminUsername: "root", adminPassword: "admin-pass",
		controllerCredential: "controller-secret", passwordParams: auth.TestParams(),
	}
}

func loginOverHTTP(t *testing.T, url, username, password string) string {
	t.Helper()
	response := call(t, http.MethodPost, url, "", map[string]string{"username": username, "password": password})
	assertStatusNotReserved(t, response, http.StatusOK)
	var body struct {
		AccessToken string `json:"accessToken"`
	}
	decodeResponse(t, response, &body)
	return body.AccessToken
}

func loginOnHandler(t *testing.T, handler http.Handler, path, username, password string) string {
	t.Helper()
	response := serve(t, handler, http.MethodPost, path, "", map[string]string{"username": username, "password": password})
	if response.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	return body.AccessToken
}

func serve(t *testing.T, handler http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var input io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode handler request: %v", err)
		}
		input = bytes.NewReader(encoded)
	}
	request := httptest.NewRequest(method, path, input)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func call(t *testing.T, method, url, token string, body any) *http.Response {
	t.Helper()
	var input io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode request: %v", err)
		}
		input = bytes.NewReader(encoded)
	}
	request, err := http.NewRequest(method, url, input)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("HTTP request: %v", err)
	}
	return response
}

func assertStatusNotReserved(t *testing.T, response *http.Response, want int) {
	t.Helper()
	if response.StatusCode == http.StatusNotImplemented {
		t.Fatalf("route still uses reserved 501 handler: %s", readAndClose(t, response))
	}
	if response.StatusCode != want {
		t.Fatalf("status=%d want=%d body=%s", response.StatusCode, want, readAndClose(t, response))
	}
}

func decodeResponse(t *testing.T, response *http.Response, target any) {
	t.Helper()
	defer response.Body.Close()
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func readAndClose(t *testing.T, response *http.Response) string {
	t.Helper()
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return string(data)
}

func TestStackModeFailsClosedWithoutExplicitKernelAndLock(t *testing.T) {
	cfg := testConfig(t)
	cfg.mode = "container-lab"
	cfg.address = "0.0.0.0:8080"
	cfg.wireGuardMode = "fake"
	if err := validateConfig(cfg); err == nil {
		t.Fatal("container lab accepted fake controller")
	}
	cfg.wireGuardMode = "kernel"
	cfg.interfaceName = "wg0"
	cfg.pool4 = "10.77.77.0/24"
	cfg.pool6 = "fd42:77:77::/64"
	cfg.endpointPort = 51443
	if err := validateConfig(cfg); err != nil {
		t.Fatal(err)
	}
	cfg.mode = "local"
	if err := validateConfig(cfg); err == nil {
		t.Fatal("local bind restriction lost")
	}
}
