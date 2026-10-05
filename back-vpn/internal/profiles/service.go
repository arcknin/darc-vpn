package profiles

import (
	"context"
	"encoding/json"

	"darc-vpn/back-vpn/internal/auth"
	profilecrypto "darc-vpn/back-vpn/internal/crypto"
	"darc-vpn/back-vpn/internal/domain"
)

type Repository interface {
	Profiles(ctx context.Context, accountID string) ([]domain.ProfileRecord, error)
	Profile(ctx context.Context, accountID, profileID string) (domain.ProfileRecord, error)
}

type AcceptanceGate interface {
	AuthorizeProfileDelivery(ctx context.Context, accountID string) error
}

type Service struct {
	repo     Repository
	cipher   domain.ProfileCipher
	gate     AcceptanceGate
	metadata func(context.Context, domain.ProfileRecord) (string, int, error)
}

func NewService(repo Repository, cipher domain.ProfileCipher, gate AcceptanceGate) *Service {
	return &Service{repo: repo, cipher: cipher, gate: gate}
}

func (s *Service) WithDeliveryMetadata(fn func(context.Context, domain.ProfileRecord) (string, int, error)) *Service {
	s.metadata = fn
	return s
}

func (s *Service) List(ctx context.Context, principal auth.Principal) ([]domain.ProfileSummary, error) {
	if principal.Role != domain.RoleFriend {
		return nil, forbidden()
	}
	if err := s.authorizeDelivery(ctx, principal.AccountID); err != nil {
		return nil, err
	}
	records, err := s.repo.Profiles(ctx, principal.AccountID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.ProfileSummary, 0, len(records))
	for _, record := range records {
		summary, err := record.Summary()
		if err != nil {
			return nil, err
		}
		result = append(result, summary)
	}
	return result, nil
}

func (s *Service) Get(ctx context.Context, principal auth.Principal, profileID string) (domain.ProfileSummary, error) {
	if principal.Role != domain.RoleFriend {
		return domain.ProfileSummary{}, forbidden()
	}
	if err := s.authorizeDelivery(ctx, principal.AccountID); err != nil {
		return domain.ProfileSummary{}, err
	}
	record, err := s.repo.Profile(ctx, principal.AccountID, profileID)
	if err != nil {
		return domain.ProfileSummary{}, profileNotFound(err)
	}
	return record.Summary()
}

func (s *Service) Recover(ctx context.Context, principal auth.Principal, profileID string) (domain.ProfilePrivate, error) {
	if principal.Role != domain.RoleFriend {
		return domain.ProfilePrivate{}, forbidden()
	}
	if err := s.authorizeDelivery(ctx, principal.AccountID); err != nil {
		return domain.ProfilePrivate{}, err
	}
	record, err := s.repo.Profile(ctx, principal.AccountID, profileID)
	if err != nil {
		return domain.ProfilePrivate{}, profileNotFound(err)
	}
	var serverKey string
	var port int
	if s.metadata != nil {
		var err error
		serverKey, port, err = s.metadata(ctx, record)
		if err != nil {
			return domain.ProfilePrivate{}, err
		}
	}
	aad := profilecrypto.AssociatedData(record.AccountID, record.ID, record.EncryptedPrivate.Version)
	plaintext, err := s.cipher.Decrypt(ctx, record.EncryptedPrivate, aad)
	if err != nil {
		return domain.ProfilePrivate{}, domain.NewCodedError(domain.ErrProfileRecoveryFailed, "private profile could not be recovered")
	}
	var payload privatePayload
	if err := json.Unmarshal(plaintext, &payload); err != nil || payload.PrivateKey == "" {
		return domain.ProfilePrivate{}, domain.NewCodedError(domain.ErrProfileRecoveryFailed, "private profile could not be recovered")
	}
	return domain.ProfilePrivate{
		ServerPublicKey: serverKey, EndpointPort: port,
		ID: record.ID, DisplayName: record.DisplayName, Endpoint: record.Endpoint, Address: record.Address,
		AllowedIPs: append([]string(nil), record.AllowedIPs...), DNS: append([]string(nil), record.DNS...),
		PeerPublicKey: record.PeerPublicKey, PrivateKey: payload.PrivateKey, PresharedKey: payload.PresharedKey,
	}, nil
}

func (s *Service) authorizeDelivery(ctx context.Context, accountID string) error {
	if s.gate == nil {
		return domain.NewCodedError(domain.ErrTermsAcceptanceRequired, "terms acceptance is required")
	}
	return s.gate.AuthorizeProfileDelivery(ctx, accountID)
}

func forbidden() error {
	return domain.NewCodedError(domain.ErrForbidden, "operation is not allowed for this role")
}

func profileNotFound(err error) error {
	_ = err
	return domain.NewCodedError(domain.ErrProfileNotFound, "profile was not found")
}
