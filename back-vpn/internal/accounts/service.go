package accounts

import (
	"context"
	"strings"

	"darc-vpn/back-vpn/internal/auth"
	"darc-vpn/back-vpn/internal/domain"
)

type Repository interface {
	CreateAdmin(ctx context.Context, account domain.AccountRecord, passwordHash string) error
	CreateAccountWithPrimary(ctx context.Context, account domain.AccountRecord, passwordHash string, profile domain.ProfileRecord) error
	AddProfile(ctx context.Context, accountID string, profile domain.ProfileRecord) error
	Account(ctx context.Context, accountID string) (domain.AccountRecord, error)
	Accounts(ctx context.Context) ([]domain.AccountRecord, error)
	Profiles(ctx context.Context, accountID string) ([]domain.ProfileRecord, error)
}

type ProfileFactory interface {
	Create(ctx context.Context, accountID, displayName string, primary bool) (domain.ProfileRecord, error)
}

type CreateRequest struct {
	Username           string `json:"username"`
	Password           string `json:"password"`
	PrimaryDisplayName string `json:"primaryDisplayName"`
}

type Service struct {
	repo        Repository
	hasher      domain.PasswordHasher
	profiles    ProfileFactory
	ids         domain.IDGenerator
	clock       domain.Clock
	afterCreate func(context.Context)
}

func NewService(repo Repository, hasher domain.PasswordHasher, profiles ProfileFactory, ids domain.IDGenerator, clock domain.Clock) *Service {
	return &Service{repo: repo, hasher: hasher, profiles: profiles, ids: ids, clock: clock}
}

func (s *Service) WithProvisioning(fn func(context.Context)) *Service { s.afterCreate = fn; return s }

func (s *Service) BootstrapAdmin(ctx context.Context, username, password string) (auth.Principal, error) {
	if strings.TrimSpace(username) == "" || password == "" {
		return auth.Principal{}, invalidRequest()
	}
	id, err := s.ids.NewID()
	if err != nil {
		return auth.Principal{}, err
	}
	hash, err := s.hasher.Hash(ctx, password)
	if err != nil {
		return auth.Principal{}, err
	}
	admin := domain.AccountRecord{ID: id, Username: strings.TrimSpace(username), Role: domain.RoleAdmin, Status: domain.AccountActive, CreatedAt: s.clock.Now().UTC()}
	if err := s.repo.CreateAdmin(ctx, admin, hash); err != nil {
		return auth.Principal{}, err
	}
	return auth.Principal{AccountID: id, Role: domain.RoleAdmin}, nil
}

func (s *Service) Create(ctx context.Context, principal auth.Principal, request CreateRequest) (domain.AdminAccountMetadata, error) {
	if principal.Role != domain.RoleAdmin {
		return domain.AdminAccountMetadata{}, forbidden()
	}
	if strings.TrimSpace(request.Username) == "" || request.Password == "" || strings.TrimSpace(request.PrimaryDisplayName) == "" {
		return domain.AdminAccountMetadata{}, invalidRequest()
	}
	accountID, err := s.ids.NewID()
	if err != nil {
		return domain.AdminAccountMetadata{}, err
	}
	profile, err := s.profiles.Create(ctx, accountID, request.PrimaryDisplayName, true)
	if err != nil {
		return domain.AdminAccountMetadata{}, err
	}
	hash, err := s.hasher.Hash(ctx, request.Password)
	if err != nil {
		return domain.AdminAccountMetadata{}, err
	}
	account := domain.AccountRecord{
		ID: accountID, Username: strings.TrimSpace(request.Username), Role: domain.RoleFriend, Status: domain.AccountActive,
		CreatedAt: s.clock.Now().UTC(), PrimaryID: profile.ID, ProfileCount: 1,
	}
	if err := s.repo.CreateAccountWithPrimary(ctx, account, hash, profile); err != nil {
		return domain.AdminAccountMetadata{}, err
	}
	if s.afterCreate != nil {
		s.afterCreate(ctx)
		return s.Get(ctx, principal, accountID)
	}
	return metadata(account, []domain.ProfileRecord{profile})
}

func (s *Service) CreateExtraProfile(ctx context.Context, principal auth.Principal, accountID, displayName string) (domain.AdminProfileMetadata, error) {
	if principal.Role != domain.RoleAdmin {
		return domain.AdminProfileMetadata{}, forbidden()
	}
	if strings.TrimSpace(displayName) == "" {
		return domain.AdminProfileMetadata{}, invalidRequest()
	}
	if _, err := s.repo.Account(ctx, accountID); err != nil {
		return domain.AdminProfileMetadata{}, err
	}
	profile, err := s.profiles.Create(ctx, accountID, displayName, false)
	if err != nil {
		return domain.AdminProfileMetadata{}, err
	}
	if err := s.repo.AddProfile(ctx, accountID, profile); err != nil {
		return domain.AdminProfileMetadata{}, err
	}
	if s.afterCreate != nil {
		s.afterCreate(ctx)
		items, err := s.repo.Profiles(ctx, accountID)
		if err != nil {
			return domain.AdminProfileMetadata{}, err
		}
		for _, p := range items {
			if p.ID == profile.ID {
				return p.AdminMetadata()
			}
		}
		return domain.AdminProfileMetadata{}, invalidRequest()
	}
	return profile.AdminMetadata()
}

func (s *Service) Get(ctx context.Context, principal auth.Principal, accountID string) (domain.AdminAccountMetadata, error) {
	if principal.Role != domain.RoleAdmin {
		return domain.AdminAccountMetadata{}, forbidden()
	}
	account, err := s.repo.Account(ctx, accountID)
	if err != nil {
		return domain.AdminAccountMetadata{}, err
	}
	profiles, err := s.repo.Profiles(ctx, accountID)
	if err != nil {
		return domain.AdminAccountMetadata{}, err
	}
	return metadata(account, profiles)
}

func (s *Service) List(ctx context.Context, principal auth.Principal) ([]domain.AdminAccountMetadata, error) {
	if principal.Role != domain.RoleAdmin {
		return nil, forbidden()
	}
	accounts, err := s.repo.Accounts(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]domain.AdminAccountMetadata, 0, len(accounts))
	for _, account := range accounts {
		profiles, err := s.repo.Profiles(ctx, account.ID)
		if err != nil {
			return nil, err
		}
		item, err := metadata(account, profiles)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func metadata(account domain.AccountRecord, profiles []domain.ProfileRecord) (domain.AdminAccountMetadata, error) {
	items := make([]domain.AdminProfileMetadata, 0, len(profiles))
	for _, profile := range profiles {
		item, err := profile.AdminMetadata()
		if err != nil {
			return domain.AdminAccountMetadata{}, err
		}
		items = append(items, item)
	}
	return domain.AdminAccountMetadata{
		ID: account.ID, Username: account.Username, Status: account.Status, CreatedAt: account.CreatedAt,
		PrimaryID: account.PrimaryID, ProfileCount: len(items), Profiles: items,
	}, nil
}

func forbidden() error {
	return domain.NewCodedError(domain.ErrForbidden, "operation is not allowed for this role")
}

func invalidRequest() error {
	return domain.NewCodedError(domain.ErrInvalidRequest, "request is invalid")
}
