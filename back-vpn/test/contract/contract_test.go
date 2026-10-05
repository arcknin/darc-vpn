package contract_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"darc-vpn/back-vpn/internal/domain"
	"darc-vpn/back-vpn/internal/platform"
	"darc-vpn/back-vpn/internal/platform/httpapi"
)

func TestOpenAPIContractDefinesApprovedSurface(t *testing.T) {
	contents, err := os.ReadFile("../../api/openapi.yaml")
	if err != nil {
		t.Fatalf("read OpenAPI contract: %v", err)
	}
	openAPI := string(contents)

	required := []string{
		"openapi: 3.0.3",
		"/api/v1/auth/friend/login:",
		"/api/v1/auth/admin/login:",
		"/api/v1/terms/current:",
		"/api/v1/terms/acceptance:",
		"/api/v1/profiles:",
		"/api/v1/profiles/{profileId}/private:",
		"/api/v1/admin/accounts:",
		"/api/v1/admin/accounts/{accountId}/profiles:",
		"/api/v1/admin/accounts/{accountId}/block:",
		"/api/v1/admin/accounts/{accountId}/revocation:",
		"vpn.darcanix.com",
		"AdminProfileMetadata:",
		"ProfilePrivate:",
		"terms_acceptance_required",
		"revocation_not_confirmed",
	}
	for _, marker := range required {
		if !strings.Contains(openAPI, marker) {
			t.Errorf("OpenAPI contract is missing %q", marker)
		}
	}
}

func TestOpenAPIKeepsExternalHostsUnspecifiedAndEndpointFixed(t *testing.T) {
	openAPI := readOpenAPI(t)
	if strings.Contains(openAPI, "servers:") {
		t.Fatal("OpenAPI must not publish an unapproved API or dashboard host")
	}
	if strings.Contains(openAPI, "dashboard-vpn.") {
		t.Fatal("dashboard-vpn FQDN is still pending and must not be invented")
	}
	if strings.Count(openAPI, "enum: [vpn.darcanix.com]") != 2 {
		t.Fatal("profile schemas must reserve vpn.darcanix.com as the only endpoint")
	}
}

func TestOpenAPIAdminSchemaHasNoPrivateFields(t *testing.T) {
	openAPI := readOpenAPI(t)
	start := strings.Index(openAPI, "    AdminProfileMetadata:\n")
	if start < 0 {
		t.Fatal("AdminProfileMetadata schema is missing")
	}
	end := strings.Index(openAPI[start+1:], "    AdminAccountMetadata:\n")
	if end < 0 {
		t.Fatal("AdminAccountMetadata schema is missing")
	}
	adminSchema := openAPI[start : start+1+end]
	for _, forbidden := range []string{"privateKey:", "presharedKey:", "configuration:"} {
		if strings.Contains(adminSchema, forbidden) {
			t.Errorf("admin DTO contains forbidden field %q", forbidden)
		}
	}
}

func TestRouteInventoryRequiresAuthenticatedTLSForProtectedOperations(t *testing.T) {
	openAPI := readOpenAPI(t)
	routes := httpapi.Routes()
	seen := make(map[string]bool, len(routes))
	for _, route := range routes {
		key := route.Method + " " + route.Path
		if seen[key] {
			t.Fatalf("duplicate route %s", key)
		}
		seen[key] = true
		if route.Path == "/healthz" {
			if !strings.Contains(openAPI, "  /healthz:\n") {
				t.Error("health route is missing from OpenAPI")
			}
			continue
		}
		pathBlock := openAPIPathBlock(openAPI, route.Path)
		if pathBlock == "" {
			t.Errorf("route %s is missing from OpenAPI", key)
			continue
		}
		if !strings.Contains(pathBlock, "    "+strings.ToLower(route.Method)+":\n") {
			t.Errorf("route %s has no matching operation in OpenAPI", key)
		}
		if route.RequiresTLS && !strings.Contains(pathBlock, "x-requires-https: true") {
			t.Errorf("route %s is missing the HTTPS contract extension", key)
		}
		if route.RequiresAcceptance && !strings.Contains(pathBlock, "x-requires-terms-acceptance: true") {
			t.Errorf("route %s is missing the terms gate contract extension", key)
		}
		roles := make([]string, 0, len(route.Roles))
		for _, role := range route.Roles {
			roles = append(roles, string(role))
		}
		if len(roles) > 0 && !strings.Contains(pathBlock, "x-allowed-roles: ["+strings.Join(roles, ", ")+"]") {
			t.Errorf("route %s is missing its authorization role extension", key)
		}
		if !route.RequiresTLS {
			t.Errorf("%s must require encrypted transport", key)
		}
		if strings.HasSuffix(route.Path, "/login") {
			if route.RequiresAuth {
				t.Errorf("%s cannot require a session before login", key)
			}
			continue
		}
		if !route.RequiresAuth {
			t.Errorf("%s must require an authenticated session", key)
		}
		if len(route.Roles) == 0 {
			t.Errorf("%s must declare an authorization role", key)
		}
	}
}

func TestHTTPAdapterExposesHealthAndReservesLaterRoutes(t *testing.T) {
	healthRequest := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	healthRecorder := httptest.NewRecorder()
	httpapi.NewHandler().ServeHTTP(healthRecorder, healthRequest)
	if healthRecorder.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d", healthRecorder.Code, http.StatusOK)
	}

	reservedRequest := httptest.NewRequest(http.MethodGet, "/api/v1/profiles", nil)
	reservedRecorder := httptest.NewRecorder()
	httpapi.NewHandler().ServeHTTP(reservedRecorder, reservedRequest)
	if reservedRecorder.Code != http.StatusNotImplemented {
		t.Fatalf("reserved route status = %d, want %d", reservedRecorder.Code, http.StatusNotImplemented)
	}
}

func TestDomainDTOBoundariesAndEndpointRule(t *testing.T) {
	profile := domain.ProfileRecord{
		ID:            "profile-1",
		AccountID:     "account-1",
		DisplayName:   "principal",
		IsPrimary:     true,
		Endpoint:      domain.VPNEndpoint,
		PeerPublicKey: "public",
		EncryptedPrivate: domain.EncryptedPrivateProfile{
			Ciphertext: []byte("ciphertext"),
			Nonce:      []byte("nonce"),
			Version:    "v1",
		},
	}
	if _, err := profile.Summary(); err != nil {
		t.Fatalf("valid profile rejected: %v", err)
	}
	admin, err := profile.AdminMetadata()
	if err != nil {
		t.Fatalf("valid admin profile rejected: %v", err)
	}
	forbiddenFields := map[string]bool{"PrivateKey": true, "PresharedKey": true, "EncryptedPrivate": true}
	typeOfAdmin := reflect.TypeOf(admin)
	for index := 0; index < typeOfAdmin.NumField(); index++ {
		if forbiddenFields[typeOfAdmin.Field(index).Name] {
			t.Fatalf("admin DTO exposes forbidden field %q", typeOfAdmin.Field(index).Name)
		}
	}
	invalidProfile := domain.ProfileRecord{Endpoint: "example.invalid"}
	if _, err := invalidProfile.Summary(); domain.CodeOf(err) != domain.ErrInvalidRequest {
		t.Fatalf("unexpected endpoint validation error: %v", err)
	}
}

func TestLocalFakesImplementB01Ports(t *testing.T) {
	var _ domain.TransactionalStore = platform.NewMemoryTransactionalStore()
	var _ domain.TermsStore = &platform.MemoryTermsStore{}
	var _ domain.AccessMetadataSink = &platform.MemoryAccessMetadataSink{}
	var _ domain.SessionStore = platform.NewMemorySessionStore()
	var _ domain.IDGenerator = &platform.SequenceIDGenerator{Prefix: "test"}
	var _ domain.Clock = platform.FixedClock{}
	var _ domain.WireGuardKeyGenerator = &platform.DeterministicKeyGenerator{}
	var _ domain.WireGuardController = &platform.FakeWireGuardController{}
}

func readOpenAPI(t *testing.T) string {
	t.Helper()
	contents, err := os.ReadFile("../../api/openapi.yaml")
	if err != nil {
		t.Fatalf("read OpenAPI contract: %v", err)
	}
	return string(contents)
}

func openAPIPathBlock(document, path string) string {
	marker := "  " + path + ":\n"
	start := strings.Index(document, marker)
	if start < 0 {
		return ""
	}
	remainder := document[start+len(marker):]
	end := len(remainder)
	for _, candidate := range []string{"\n  /", "\ncomponents:"} {
		if index := strings.Index(remainder, candidate); index >= 0 && index < end {
			end = index
		}
	}
	return remainder[:end]
}

func TestStackContractDefinesServerIdentityReadinessAndProvisioning(t *testing.T) {
	doc := readOpenAPI(t)
	for _, marker := range []string{"serverPublicKey:", "endpointPort:", "provisioningState:", "profile_not_ready", "pool_exhausted", "  /readyz:"} {
		if !strings.Contains(doc, marker) {
			t.Errorf("missing stack contract %s", marker)
		}
	}
}
func TestStackComposeBoundary(t *testing.T) {
	data, err := os.ReadFile("../../../infra/wireguard-stack/compose.yaml")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, v := range []string{"network_mode: service:wireguard", "127.0.0.1:", "NET_ADMIN", "cap_drop: [ALL]"} {
		if !strings.Contains(s, v) {
			t.Errorf("missing compose boundary %s", v)
		}
	}
	for _, v := range []string{"docker.sock", "privileged: true", "network_mode: host", "SYS_ADMIN", "SYS_MODULE", "NET_RAW"} {
		if strings.Contains(s, v) {
			t.Errorf("forbidden permission %s", v)
		}
	}
}
