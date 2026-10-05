package revocation

import (
	"context"
	"strings"
	"sync"
	"time"

	"darc-vpn/back-vpn/internal/auth"
	"darc-vpn/back-vpn/internal/domain"
)

type Repository interface {
	BeginAccountBlock(context.Context, string, time.Time) (domain.RevocationOperation, bool, error)
	SaveRevocation(context.Context, domain.RevocationOperation) error
	Revocation(context.Context, string) (domain.RevocationOperation, error)
}

type Service struct {
	repo       Repository
	controller domain.WireGuardController
	clock      domain.Clock
	blocks     sync.Mutex
}

func NewService(repo Repository, controller domain.WireGuardController, clock domain.Clock) *Service {
	return &Service{repo: repo, controller: controller, clock: clock}
}

func (s *Service) Block(ctx context.Context, principal auth.Principal, accountID string) (domain.RevocationOperation, error) {
	if principal.Role != domain.RoleAdmin {
		return domain.RevocationOperation{}, forbidden()
	}
	if strings.TrimSpace(accountID) == "" {
		return domain.RevocationOperation{}, invalidRequest()
	}

	// Serializing the local command keeps duplicate HTTP requests from issuing
	// overlapping controller calls. The database primary key remains the
	// durable guard across service instances and process restarts.
	s.blocks.Lock()
	defer s.blocks.Unlock()

	operation, _, err := s.repo.BeginAccountBlock(ctx, accountID, s.clock.Now().UTC())
	if err != nil {
		return domain.RevocationOperation{}, err
	}
	if operation.Status == domain.RevocationCompleted {
		return operation, nil
	}

	peerIDs := make([]string, 0, len(operation.Peers))
	for _, peer := range operation.Peers {
		peerIDs = append(peerIDs, peer.PeerID)
	}
	operation = pendingSnapshot(operation, s.clock)
	removals, removalErr := s.controller.RemovePeers(ctx, peerIDs)
	if removalErr != nil {
		operation.Status = domain.RevocationFailed
		operation.Error = "peer_removal_failed"
		for index := range operation.Peers {
			operation.Peers[index].Error = "removal_failed"
		}
		if err := s.repo.SaveRevocation(ctx, operation); err != nil {
			return domain.RevocationOperation{}, err
		}
		return operation, nil
	}

	activeConfirmed, activeFailed := applyRemovalResults(&operation, removals)
	if activeFailed {
		operation.Status = domain.RevocationFailed
		operation.Error = "peer_removal_failed"
		if err := s.repo.SaveRevocation(ctx, operation); err != nil {
			return domain.RevocationOperation{}, err
		}
		return operation, nil
	}
	if !activeConfirmed {
		operation.Status = domain.RevocationPending
		operation.Error = "peer_removal_pending"
		if err := s.repo.SaveRevocation(ctx, operation); err != nil {
			return domain.RevocationOperation{}, err
		}
		return operation, nil
	}

	persistentConfirmed, confirmationErr := s.controller.ConfirmPeersRemoved(ctx, peerIDs)
	switch {
	case confirmationErr != nil:
		operation.Status = domain.RevocationFailed
		operation.Error = "persistent_confirmation_failed"
	case !persistentConfirmed:
		operation.Status = domain.RevocationPending
		operation.Error = "persistent_confirmation_pending"
	default:
		operation.Status = domain.RevocationCompleted
		operation.Error = ""
	}
	if err := s.repo.SaveRevocation(ctx, operation); err != nil {
		return domain.RevocationOperation{}, err
	}
	return operation, nil
}

func (s *Service) Get(ctx context.Context, principal auth.Principal, accountID string) (domain.RevocationOperation, error) {
	if principal.Role != domain.RoleAdmin {
		return domain.RevocationOperation{}, forbidden()
	}
	if strings.TrimSpace(accountID) == "" {
		return domain.RevocationOperation{}, invalidRequest()
	}
	return s.repo.Revocation(ctx, accountID)
}

func pendingSnapshot(operation domain.RevocationOperation, clock domain.Clock) domain.RevocationOperation {
	operation.Status = domain.RevocationPending
	operation.Error = ""
	operation.UpdatedAt = clock.Now().UTC()
	for index := range operation.Peers {
		operation.Peers[index].Removed = false
		operation.Peers[index].Error = ""
	}
	return operation
}

func applyRemovalResults(operation *domain.RevocationOperation, removals []domain.WireGuardRemoval) (bool, bool) {
	byPeer := make(map[string]domain.WireGuardRemoval, len(removals))
	duplicates := make(map[string]bool)
	for _, removal := range removals {
		if _, exists := byPeer[removal.PeerID]; exists {
			duplicates[removal.PeerID] = true
		}
		byPeer[removal.PeerID] = removal
	}
	allRemoved := len(operation.Peers) > 0
	anyFailure := false
	for index := range operation.Peers {
		peer := &operation.Peers[index]
		removal, exists := byPeer[peer.PeerID]
		switch {
		case !exists || duplicates[peer.PeerID]:
			peer.Error = "removal_unconfirmed"
			allRemoved = false
		case removal.Error != "":
			peer.Error = "removal_failed"
			allRemoved = false
			anyFailure = true
		case !removal.Removed:
			peer.Error = "removal_unconfirmed"
			allRemoved = false
		default:
			peer.Removed = true
			peer.Error = ""
		}
	}
	return allRemoved, anyFailure
}

func forbidden() error {
	return domain.NewCodedError(domain.ErrForbidden, "operation is not allowed for this role")
}

func invalidRequest() error {
	return domain.NewCodedError(domain.ErrInvalidRequest, "request is invalid")
}
