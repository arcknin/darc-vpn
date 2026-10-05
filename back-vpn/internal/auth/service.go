package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"time"

	"darc-vpn/back-vpn/internal/domain"
)

var ErrCredentialNotFound = errors.New("credential not found")

type Credential struct {
	AccountID    string
	PasswordHash string
	Role         domain.Role
	Status       domain.AccountStatus
}

type StoredSession struct {
	ID          string
	TokenDigest []byte
	AccountID   string
	Role        domain.Role
	ExpiresAt   time.Time
	CreatedAt   time.Time
	Status      domain.AccountStatus
}

type Repository interface {
	CredentialByUsername(ctx context.Context, username string, role domain.Role) (Credential, error)
	SaveSession(ctx context.Context, session StoredSession) error
	SessionByDigest(ctx context.Context, digest []byte) (StoredSession, error)
	DeleteSessionByDigest(ctx context.Context, digest []byte) error
}

type Principal struct {
	AccountID string
	Role      domain.Role
}

type LoginResponse struct {
	AccessToken string      `json:"accessToken"`
	ExpiresAt   time.Time   `json:"expiresAt"`
	AccountID   string      `json:"accountId"`
	Role        domain.Role `json:"role"`
}

type Config struct {
	SessionTTL time.Duration
	Random     io.Reader
}

type Service struct {
	repo      Repository
	hasher    domain.PasswordHasher
	clock     domain.Clock
	ttl       time.Duration
	random    io.Reader
	dummyHash string
}

func NewService(repo Repository, hasher domain.PasswordHasher, clock domain.Clock, config Config) (*Service, error) {
	if config.SessionTTL <= 0 {
		return nil, errors.New("session TTL must be positive")
	}
	if config.Random == nil {
		config.Random = rand.Reader
	}
	dummy, err := hasher.Hash(context.Background(), "invalid credential sentinel")
	if err != nil {
		return nil, err
	}
	return &Service{repo: repo, hasher: hasher, clock: clock, ttl: config.SessionTTL, random: config.Random, dummyHash: dummy}, nil
}

func (s *Service) Login(ctx context.Context, username, password string, role domain.Role) (LoginResponse, error) {
	if err := domain.ValidateRole(role); err != nil {
		return LoginResponse{}, invalidCredentials()
	}
	credential, findErr := s.repo.CredentialByUsername(ctx, username, role)
	hash := credential.PasswordHash
	if findErr != nil {
		hash = s.dummyHash
	}
	match, verifyErr := s.hasher.Verify(ctx, hash, password)
	if findErr != nil || verifyErr != nil || !match {
		return LoginResponse{}, invalidCredentials()
	}
	if credential.Status == domain.AccountBlocked {
		return LoginResponse{}, domain.NewCodedError(domain.ErrAccountBlocked, "account is blocked")
	}
	tokenBytes := make([]byte, 32)
	if _, err := io.ReadFull(s.random, tokenBytes); err != nil {
		return LoginResponse{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	digest := sha256.Sum256([]byte(token))
	now := s.clock.Now().UTC()
	session := StoredSession{
		ID:          base64.RawURLEncoding.EncodeToString(digest[:16]),
		TokenDigest: digest[:],
		AccountID:   credential.AccountID,
		Role:        credential.Role,
		ExpiresAt:   now.Add(s.ttl),
		CreatedAt:   now,
	}
	if err := s.repo.SaveSession(ctx, session); err != nil {
		return LoginResponse{}, err
	}
	return LoginResponse{AccessToken: token, ExpiresAt: session.ExpiresAt, AccountID: session.AccountID, Role: session.Role}, nil
}

func (s *Service) Authenticate(ctx context.Context, token string, roles ...domain.Role) (Principal, error) {
	if token == "" {
		return Principal{}, notAuthenticated()
	}
	digest := sha256.Sum256([]byte(token))
	session, err := s.repo.SessionByDigest(ctx, digest[:])
	if err != nil || !s.clock.Now().Before(session.ExpiresAt) {
		return Principal{}, notAuthenticated()
	}
	if session.Status == domain.AccountBlocked {
		return Principal{}, domain.NewCodedError(domain.ErrAccountBlocked, "account is blocked")
	}
	for _, role := range roles {
		if session.Role == role {
			return Principal{AccountID: session.AccountID, Role: session.Role}, nil
		}
	}
	return Principal{}, domain.NewCodedError(domain.ErrForbidden, "operation is not allowed for this role")
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return notAuthenticated()
	}
	digest := sha256.Sum256([]byte(token))
	if err := s.repo.DeleteSessionByDigest(ctx, digest[:]); err != nil {
		return notAuthenticated()
	}
	return nil
}

func invalidCredentials() error {
	return domain.NewCodedError(domain.ErrInvalidCredentials, "invalid credentials")
}

func notAuthenticated() error {
	return domain.NewCodedError(domain.ErrNotAuthenticated, "session is not authenticated")
}
