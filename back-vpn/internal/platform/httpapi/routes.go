package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"darc-vpn/back-vpn/internal/domain"
)

type Route struct {
	Name               string
	Method             string
	Path               string
	Roles              []domain.Role
	RequiresTLS        bool
	RequiresAuth       bool
	RequiresAcceptance bool
}

// Routes is the executable route inventory corresponding to api/openapi.yaml.
// Business behavior is added by later batches; B01 freezes method, path and
// authorization boundaries without inventing live integrations.
func Routes() []Route {
	return []Route{
		{Name: "health", Method: http.MethodGet, Path: "/healthz"},
		{Name: "friend_login", Method: http.MethodPost, Path: "/api/v1/auth/friend/login", Roles: []domain.Role{domain.RoleFriend}, RequiresTLS: true},
		{Name: "admin_login", Method: http.MethodPost, Path: "/api/v1/auth/admin/login", Roles: []domain.Role{domain.RoleAdmin}, RequiresTLS: true},
		{Name: "logout", Method: http.MethodPost, Path: "/api/v1/auth/logout", Roles: []domain.Role{domain.RoleFriend, domain.RoleAdmin}, RequiresTLS: true, RequiresAuth: true},
		{Name: "current_terms", Method: http.MethodGet, Path: "/api/v1/terms/current", Roles: []domain.Role{domain.RoleFriend}, RequiresTLS: true, RequiresAuth: true},
		{Name: "accept_terms", Method: http.MethodPost, Path: "/api/v1/terms/acceptance", Roles: []domain.Role{domain.RoleFriend}, RequiresTLS: true, RequiresAuth: true},
		{Name: "list_profiles", Method: http.MethodGet, Path: "/api/v1/profiles", Roles: []domain.Role{domain.RoleFriend}, RequiresTLS: true, RequiresAuth: true, RequiresAcceptance: true},
		{Name: "get_profile", Method: http.MethodGet, Path: "/api/v1/profiles/{profileId}", Roles: []domain.Role{domain.RoleFriend}, RequiresTLS: true, RequiresAuth: true, RequiresAcceptance: true},
		{Name: "get_private_profile", Method: http.MethodGet, Path: "/api/v1/profiles/{profileId}/private", Roles: []domain.Role{domain.RoleFriend}, RequiresTLS: true, RequiresAuth: true, RequiresAcceptance: true},
		{Name: "list_accounts", Method: http.MethodGet, Path: "/api/v1/admin/accounts", Roles: []domain.Role{domain.RoleAdmin}, RequiresTLS: true, RequiresAuth: true},
		{Name: "create_account", Method: http.MethodPost, Path: "/api/v1/admin/accounts", Roles: []domain.Role{domain.RoleAdmin}, RequiresTLS: true, RequiresAuth: true},
		{Name: "get_account", Method: http.MethodGet, Path: "/api/v1/admin/accounts/{accountId}", Roles: []domain.Role{domain.RoleAdmin}, RequiresTLS: true, RequiresAuth: true},
		{Name: "create_profile", Method: http.MethodPost, Path: "/api/v1/admin/accounts/{accountId}/profiles", Roles: []domain.Role{domain.RoleAdmin}, RequiresTLS: true, RequiresAuth: true},
		{Name: "block_account", Method: http.MethodPost, Path: "/api/v1/admin/accounts/{accountId}/block", Roles: []domain.Role{domain.RoleAdmin}, RequiresTLS: true, RequiresAuth: true},
		{Name: "get_revocation", Method: http.MethodGet, Path: "/api/v1/admin/accounts/{accountId}/revocation", Roles: []domain.Role{domain.RoleAdmin}, RequiresTLS: true, RequiresAuth: true},
	}
}

// NewHandler exposes only the health probe and reserved route surface. A
// reserved endpoint returns 501 until its owning batch wires the use case;
// this keeps B01 honest about what has and has not been implemented.
func NewHandler() http.Handler {
	return &router{routes: Routes()}
}

type router struct {
	routes []Route
}

func (r *router) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method == http.MethodGet && request.URL.Path == "/healthz" {
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	for _, route := range r.routes {
		if route.Name == "health" || route.Method != request.Method || !matches(route.Path, request.URL.Path) {
			continue
		}
		http.Error(writer, "route reserved for a later batch", http.StatusNotImplemented)
		return
	}
	http.NotFound(writer, request)
}

func matches(pattern, path string) bool {
	patternParts := splitPath(pattern)
	pathParts := splitPath(path)
	if len(patternParts) != len(pathParts) {
		return false
	}
	for index := range patternParts {
		if strings.HasPrefix(patternParts[index], "{") && strings.HasSuffix(patternParts[index], "}") {
			continue
		}
		if patternParts[index] != pathParts[index] {
			return false
		}
	}
	return true
}

func splitPath(value string) []string {
	trimmed := strings.Trim(value, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
