package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"darc-vpn/back-vpn/internal/accounts"
	"darc-vpn/back-vpn/internal/audit"
	"darc-vpn/back-vpn/internal/auth"
	"darc-vpn/back-vpn/internal/domain"
	"darc-vpn/back-vpn/internal/terms"
)

type AuthService interface {
	Login(context.Context, string, string, domain.Role) (auth.LoginResponse, error)
	Authenticate(context.Context, string, ...domain.Role) (auth.Principal, error)
	Logout(context.Context, string) error
}

type AccountService interface {
	Create(context.Context, auth.Principal, accounts.CreateRequest) (domain.AdminAccountMetadata, error)
	CreateExtraProfile(context.Context, auth.Principal, string, string) (domain.AdminProfileMetadata, error)
	Get(context.Context, auth.Principal, string) (domain.AdminAccountMetadata, error)
	List(context.Context, auth.Principal) ([]domain.AdminAccountMetadata, error)
}

type ProfileService interface {
	List(context.Context, auth.Principal) ([]domain.ProfileSummary, error)
	Get(context.Context, auth.Principal, string) (domain.ProfileSummary, error)
	Recover(context.Context, auth.Principal, string) (domain.ProfilePrivate, error)
}

type TermsService interface {
	Current(context.Context, auth.Principal) (domain.TermsDocument, error)
	Accept(context.Context, auth.Principal, terms.AcceptanceRequest) (domain.TermsAcceptanceReceipt, error)
}

type RevocationService interface {
	Block(context.Context, auth.Principal, string) (domain.RevocationOperation, error)
	Get(context.Context, auth.Principal, string) (domain.RevocationOperation, error)
}

type AuditRecorder interface {
	Record(context.Context, audit.Event) error
}

type API struct {
	auth     AuthService
	accounts AccountService
	profiles ProfileService
	terms    TermsService
	revoke   RevocationService
	audit    AuditRecorder
	ip       *audit.SourceIPResolver
}

type Option func(*API)

func WithTerms(service TermsService) Option {
	return func(api *API) { api.terms = service }
}

func WithRevocation(service RevocationService) Option {
	return func(api *API) { api.revoke = service }
}

func WithAudit(recorder AuditRecorder, resolver *audit.SourceIPResolver) Option {
	return func(api *API) {
		api.audit = recorder
		api.ip = resolver
	}
}

func New(authService AuthService, accountService AccountService, profileService ProfileService, options ...Option) http.Handler {
	api := &API{auth: authService, accounts: accountService, profiles: profileService}
	for _, option := range options {
		option(api)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", api.health)
	mux.HandleFunc("POST /api/v1/auth/friend/login", api.login(domain.RoleFriend))
	mux.HandleFunc("POST /api/v1/auth/admin/login", api.login(domain.RoleAdmin))
	mux.HandleFunc("POST /api/v1/auth/logout", api.logout)
	mux.HandleFunc("GET /api/v1/terms/current", api.currentTerms)
	mux.HandleFunc("POST /api/v1/terms/acceptance", api.acceptTerms)
	mux.HandleFunc("GET /api/v1/profiles", api.listProfiles)
	mux.HandleFunc("GET /api/v1/profiles/{profileId}", api.getProfile)
	mux.HandleFunc("GET /api/v1/profiles/{profileId}/private", api.recoverProfile)
	mux.HandleFunc("GET /api/v1/admin/accounts", api.listAccounts)
	mux.HandleFunc("POST /api/v1/admin/accounts", api.createAccount)
	mux.HandleFunc("GET /api/v1/admin/accounts/{accountId}", api.getAccount)
	mux.HandleFunc("POST /api/v1/admin/accounts/{accountId}/profiles", api.createExtraProfile)
	mux.HandleFunc("POST /api/v1/admin/accounts/{accountId}/block", api.blockAccount)
	mux.HandleFunc("GET /api/v1/admin/accounts/{accountId}/revocation", api.getRevocation)
	return mux
}

func (a *API) health(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (a *API) login(role domain.Role) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		var input loginRequest
		if err := decodeJSON(request, &input); err != nil {
			if auditErr := a.record(request, audit.Event{Type: "authentication"}); auditErr != nil {
				writeError(writer, auditErr)
				return
			}
			writeError(writer, domain.NewCodedError(domain.ErrInvalidRequest, "request is invalid"))
			return
		}
		response, err := a.auth.Login(request.Context(), input.Username, input.Password, role)
		accountID := ""
		if err == nil {
			accountID = response.AccountID
		}
		if auditErr := a.record(request, audit.Event{Type: "authentication", AccountID: accountID}); auditErr != nil {
			if err == nil {
				_ = a.auth.Logout(request.Context(), response.AccessToken)
			}
			writeError(writer, auditErr)
			return
		}
		if err != nil {
			writeError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, response)
	}
}

func (a *API) logout(writer http.ResponseWriter, request *http.Request) {
	token := bearerToken(request)
	principal, ok := a.principal(writer, request, domain.RoleFriend, domain.RoleAdmin)
	if !ok {
		return
	}
	_ = principal
	if err := a.auth.Logout(request.Context(), token); err != nil {
		writeError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (a *API) currentTerms(writer http.ResponseWriter, request *http.Request) {
	principal, ok := a.principal(writer, request, domain.RoleFriend)
	if !ok {
		return
	}
	if a.terms == nil {
		writeError(writer, errorsNewInternal())
		return
	}
	document, err := a.terms.Current(request.Context(), principal)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, document)
}

func (a *API) acceptTerms(writer http.ResponseWriter, request *http.Request) {
	principal, ok := a.principal(writer, request, domain.RoleFriend)
	if !ok {
		return
	}
	var input terms.AcceptanceRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, domain.NewCodedError(domain.ErrInvalidRequest, "request is invalid"))
		return
	}
	if a.terms == nil {
		writeError(writer, errorsNewInternal())
		return
	}
	receipt, err := a.terms.Accept(request.Context(), principal, input)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, receipt)
}

func (a *API) listProfiles(writer http.ResponseWriter, request *http.Request) {
	principal, ok := a.principal(writer, request, domain.RoleFriend)
	if !ok {
		return
	}
	profiles, err := a.profiles.List(request.Context(), principal)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"profiles": profiles})
}

func (a *API) getProfile(writer http.ResponseWriter, request *http.Request) {
	principal, ok := a.principal(writer, request, domain.RoleFriend)
	if !ok {
		return
	}
	profile, err := a.profiles.Get(request.Context(), principal, request.PathValue("profileId"))
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, profile)
}

func (a *API) recoverProfile(writer http.ResponseWriter, request *http.Request) {
	principal, ok := a.principal(writer, request, domain.RoleFriend)
	if !ok {
		return
	}
	profile, err := a.profiles.Recover(request.Context(), principal, request.PathValue("profileId"))
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, profile)
}

func (a *API) listAccounts(writer http.ResponseWriter, request *http.Request) {
	principal, ok := a.principal(writer, request, domain.RoleAdmin)
	if !ok {
		return
	}
	items, err := a.accounts.List(request.Context(), principal)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"accounts": items})
}

func (a *API) createAccount(writer http.ResponseWriter, request *http.Request) {
	principal, ok := a.principal(writer, request, domain.RoleAdmin)
	if !ok {
		return
	}
	var input accounts.CreateRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, domain.NewCodedError(domain.ErrInvalidRequest, "request is invalid"))
		return
	}
	created, err := a.accounts.Create(request.Context(), principal, input)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, created)
}

func (a *API) getAccount(writer http.ResponseWriter, request *http.Request) {
	principal, ok := a.principal(writer, request, domain.RoleAdmin)
	if !ok {
		return
	}
	account, err := a.accounts.Get(request.Context(), principal, request.PathValue("accountId"))
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, account)
}

type createProfileRequest struct {
	DisplayName string `json:"displayName"`
}

func (a *API) createExtraProfile(writer http.ResponseWriter, request *http.Request) {
	principal, ok := a.principal(writer, request, domain.RoleAdmin)
	if !ok {
		return
	}
	var input createProfileRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, domain.NewCodedError(domain.ErrInvalidRequest, "request is invalid"))
		return
	}
	created, err := a.accounts.CreateExtraProfile(request.Context(), principal, request.PathValue("accountId"), input.DisplayName)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, created)
}

func (a *API) blockAccount(writer http.ResponseWriter, request *http.Request) {
	principal, ok := a.principal(writer, request, domain.RoleAdmin)
	if !ok {
		return
	}
	if a.revoke == nil {
		writeError(writer, errorsNewInternal())
		return
	}
	operation, err := a.revoke.Block(request.Context(), principal, request.PathValue("accountId"))
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusAccepted, operation)
}

func (a *API) getRevocation(writer http.ResponseWriter, request *http.Request) {
	principal, ok := a.principal(writer, request, domain.RoleAdmin)
	if !ok {
		return
	}
	if a.revoke == nil {
		writeError(writer, errorsNewInternal())
		return
	}
	operation, err := a.revoke.Get(request.Context(), principal, request.PathValue("accountId"))
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, operation)
}

func (a *API) principal(writer http.ResponseWriter, request *http.Request, roles ...domain.Role) (auth.Principal, bool) {
	principal, err := a.auth.Authenticate(request.Context(), bearerToken(request), roles...)
	if err != nil {
		writeError(writer, err)
		return auth.Principal{}, false
	}
	if err := a.record(request, audit.Event{Type: "authenticated_call", AccountID: principal.AccountID, ProfileID: request.PathValue("profileId")}); err != nil {
		writeError(writer, err)
		return auth.Principal{}, false
	}
	return principal, true
}

func (a *API) record(request *http.Request, event audit.Event) error {
	if a.audit == nil {
		return nil
	}
	if a.ip != nil {
		event.SourceIP = a.ip.Resolve(request)
	}
	return a.audit.Record(request.Context(), event)
}

func bearerToken(request *http.Request) string {
	parts := strings.Fields(request.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func decodeJSON(request *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return io.ErrUnexpectedEOF
		}
		return err
	}
	return nil
}

type errorResponse struct {
	Code    domain.ErrorCode `json:"code"`
	Message string           `json:"message"`
}

func writeError(writer http.ResponseWriter, err error) {
	code := domain.CodeOf(err)
	status := http.StatusInternalServerError
	message := "internal error"
	switch code {
	case domain.ErrInvalidCredentials, domain.ErrNotAuthenticated:
		status = http.StatusUnauthorized
		message = "authentication failed"
	case domain.ErrForbidden:
		status = http.StatusForbidden
		message = "operation is forbidden"
	case domain.ErrAccountBlocked:
		status = http.StatusLocked
		message = "account is blocked"
	case domain.ErrProfileNotFound:
		status = http.StatusNotFound
		message = "profile was not found"
	case domain.ErrTermsAcceptanceRequired:
		status = http.StatusPreconditionRequired
		message = "terms acceptance is required"
	case domain.ErrInvalidRequest:
		status = http.StatusBadRequest
		message = "request is invalid"
	case domain.ErrProfileRecoveryFailed:
		message = "private profile could not be recovered"
	case domain.ErrAcceptancePersistence:
		message = "terms acceptance could not be persisted"
	case domain.ErrProfileNotReady:
		status = http.StatusServiceUnavailable
		message = "profile is not ready"
	case domain.ErrPoolExhausted:
		status = http.StatusConflict
		message = "address pool exhausted"
	case domain.ErrRevocationNotConfirmed:
		status = http.StatusConflict
		message = "revocation is not confirmed"
	}
	writeJSON(writer, status, errorResponse{Code: code, Message: message})
}

func errorsNewInternal() error {
	return domain.NewCodedError("", "internal error")
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
