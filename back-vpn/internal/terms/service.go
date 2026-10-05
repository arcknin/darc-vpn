package terms

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"unicode/utf8"

	"darc-vpn/back-vpn/internal/auth"
	"darc-vpn/back-vpn/internal/domain"
)

type AcceptanceRequest struct {
	Accepted bool   `json:"accepted"`
	Version  string `json:"version"`
	SHA256   string `json:"sha256"`
}

type Repository interface {
	PersistAcceptance(context.Context, domain.TermsAcceptanceReceipt) error
	HasAcceptance(context.Context, string, string, string) (bool, error)
}

type ReceiptStore interface {
	Persist(context.Context, domain.TermsAcceptanceReceipt) error
}

type Service struct {
	document domain.TermsDocument
	repo     Repository
	receipts ReceiptStore
	clock    domain.Clock
	ids      domain.IDGenerator
}

func NewDocument(version string, contents []byte, expectedSHA256 string) (domain.TermsDocument, error) {
	version = strings.TrimSpace(version)
	expectedSHA256 = strings.TrimSpace(expectedSHA256)
	if version == "" || len(contents) == 0 || !utf8.Valid(contents) || len(expectedSHA256) != sha256.Size*2 || expectedSHA256 != strings.ToLower(expectedSHA256) {
		return domain.TermsDocument{}, errors.New("terms configuration is incomplete")
	}
	if _, err := hex.DecodeString(expectedSHA256); err != nil {
		return domain.TermsDocument{}, errors.New("terms SHA-256 is invalid")
	}
	text := string(contents)
	for _, marker := range []string{
		"[DEFINIR]", "[E-MAIL]", "[NOME OU RAZÃO SOCIAL]", "[PRAZO A DEFINIR]",
		"Rascunho para revisão antes da publicação",
	} {
		if strings.Contains(text, marker) {
			return domain.TermsDocument{}, errors.New("terms document contains unresolved editorial fields")
		}
	}
	digest := sha256.Sum256(contents)
	actual := hex.EncodeToString(digest[:])
	if actual != expectedSHA256 {
		return domain.TermsDocument{}, errors.New("terms SHA-256 does not match the configured document")
	}
	return domain.TermsDocument{AcceptanceRequired: true, Version: version, Text: text, SHA256: actual}, nil
}

func NewService(document domain.TermsDocument, repo Repository, receipts ReceiptStore, clock domain.Clock, ids domain.IDGenerator) (*Service, error) {
	if document.Version == "" || document.Text == "" || document.SHA256 == "" || repo == nil || receipts == nil || clock == nil || ids == nil {
		return nil, errors.New("terms service configuration is incomplete")
	}
	validated, err := NewDocument(document.Version, []byte(document.Text), document.SHA256)
	if err != nil {
		return nil, err
	}
	return &Service{document: validated, repo: repo, receipts: receipts, clock: clock, ids: ids}, nil
}

func (s *Service) Current(_ context.Context, principal auth.Principal) (domain.TermsDocument, error) {
	if principal.Role != domain.RoleFriend {
		return domain.TermsDocument{}, domain.NewCodedError(domain.ErrForbidden, "operation is not allowed for this role")
	}
	return s.document, nil
}

func (s *Service) Accept(ctx context.Context, principal auth.Principal, request AcceptanceRequest) (domain.TermsAcceptanceReceipt, error) {
	if principal.Role != domain.RoleFriend {
		return domain.TermsAcceptanceReceipt{}, domain.NewCodedError(domain.ErrForbidden, "operation is not allowed for this role")
	}
	if !request.Accepted || request.Version != s.document.Version || request.SHA256 != s.document.SHA256 {
		return domain.TermsAcceptanceReceipt{}, domain.NewCodedError(domain.ErrInvalidRequest, "acceptance does not match current terms")
	}
	eventID, err := s.ids.NewID()
	if err != nil {
		return domain.TermsAcceptanceReceipt{}, persistenceError(err)
	}
	receipt := domain.TermsAcceptanceReceipt{
		AccountID:  principal.AccountID,
		Version:    s.document.Version,
		SHA256:     s.document.SHA256,
		Accepted:   true,
		AcceptedAt: s.clock.Now().UTC(),
		EventID:    eventID,
	}
	if err := s.receipts.Persist(ctx, receipt); err != nil {
		return domain.TermsAcceptanceReceipt{}, persistenceError(err)
	}
	if err := s.repo.PersistAcceptance(ctx, receipt); err != nil {
		return domain.TermsAcceptanceReceipt{}, persistenceError(err)
	}
	return receipt, nil
}

func (s *Service) AuthorizeProfileDelivery(ctx context.Context, accountID string) error {
	accepted, err := s.repo.HasAcceptance(ctx, accountID, s.document.Version, s.document.SHA256)
	if err != nil {
		return err
	}
	if !accepted {
		return domain.NewCodedError(domain.ErrTermsAcceptanceRequired, "terms acceptance is required")
	}
	return nil
}

func persistenceError(err error) error {
	return domain.WrapCodedError(domain.ErrAcceptancePersistence, "terms acceptance could not be persisted", err)
}

func (s *Service) WithAcceptanceRequired(required bool) *Service {
	s.document.AcceptanceRequired = required
	return s
}
