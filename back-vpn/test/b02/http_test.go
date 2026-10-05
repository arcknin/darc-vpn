package b02_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"darc-vpn/back-vpn/internal/accounts"
	"darc-vpn/back-vpn/internal/domain"
	httpapi "darc-vpn/back-vpn/internal/http"
)

func TestHTTPAuthorizationAndAdminDTOStaySecretFree(t *testing.T) {
	ctx := context.Background()
	store := openStore(t, filepath.Join(t.TempDir(), "http.db"))
	services := newServices(t, store, bytes32(6))
	if _, err := services.accounts.BootstrapAdmin(ctx, "root", "admin-pass"); err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}
	handler := httpapi.New(services.auth, services.accounts, services.profiles)
	adminToken := loginHTTP(t, handler, "/api/v1/auth/admin/login", "root", "admin-pass")

	create := requestJSON(t, handler, http.MethodPost, "/api/v1/admin/accounts", adminToken, accounts.CreateRequest{
		Username: "friend", Password: "friend-pass", PrimaryDisplayName: "Principal",
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create account status=%d body=%s", create.Code, create.Body.String())
	}
	body := strings.ToLower(create.Body.String())
	for _, forbidden := range []string{"privatekey", "presharedkey", "ciphertext", "configuration"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("admin response leaked %q: %s", forbidden, body)
		}
	}
	var account domain.AdminAccountMetadata
	if err := json.Unmarshal(create.Body.Bytes(), &account); err != nil {
		t.Fatalf("decode account: %v", err)
	}

	friendToken := loginHTTP(t, handler, "/api/v1/auth/friend/login", "friend", "friend-pass")
	denied := requestJSON(t, handler, http.MethodPost, "/api/v1/admin/accounts/"+account.ID+"/profiles", friendToken, map[string]string{"displayName": "Extra"})
	if denied.Code != http.StatusForbidden {
		t.Fatalf("friend admin operation status=%d body=%s", denied.Code, denied.Body.String())
	}
	unknown := requestJSON(t, handler, http.MethodGet, "/api/v1/profiles", "unknown-token", nil)
	if unknown.Code != http.StatusUnauthorized {
		t.Fatalf("unknown session status=%d body=%s", unknown.Code, unknown.Body.String())
	}
	if err := store.SetAccountStatus(ctx, account.ID, domain.AccountBlocked); err != nil {
		t.Fatalf("block account: %v", err)
	}
	blocked := requestJSON(t, handler, http.MethodGet, "/api/v1/profiles", friendToken, nil)
	if blocked.Code != http.StatusLocked {
		t.Fatalf("blocked session status=%d body=%s", blocked.Code, blocked.Body.String())
	}
}

func loginHTTP(t *testing.T, handler http.Handler, path, username, password string) string {
	t.Helper()
	response := requestJSON(t, handler, http.MethodPost, path, "", map[string]string{"username": username, "password": password})
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

func requestJSON(t *testing.T, handler http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var encoded bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&encoded).Encode(body); err != nil {
			t.Fatalf("encode request: %v", err)
		}
	}
	request := httptest.NewRequest(method, path, &encoded)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}
