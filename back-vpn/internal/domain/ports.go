package domain

import (
	"context"
	"time"
)

// TransactionalStore is the persistence port. Implementations must commit all
// changes from fn atomically or leave the store unchanged.
type TransactionalStore interface {
	WithinTransaction(ctx context.Context, fn func(context.Context, StoreTx) error) error
}

type StoreTx interface {
	Account(ctx context.Context, accountID string) (AccountRecord, error)
	AccountByUsername(ctx context.Context, username string) (AccountRecord, error)
	SaveAccount(ctx context.Context, account AccountRecord) error
	Profiles(ctx context.Context, accountID string) ([]ProfileRecord, error)
	Profile(ctx context.Context, accountID, profileID string) (ProfileRecord, error)
	SaveProfile(ctx context.Context, profile ProfileRecord) error
}

type ProfileCipher interface {
	Encrypt(ctx context.Context, plaintext []byte, associatedData []byte) (EncryptedPrivateProfile, error)
	Decrypt(ctx context.Context, encrypted EncryptedPrivateProfile, associatedData []byte) ([]byte, error)
}

type PasswordHasher interface {
	Hash(ctx context.Context, plaintext string) (encoded string, err error)
	Verify(ctx context.Context, encoded, plaintext string) (match bool, err error)
}

type SessionStore interface {
	Issue(ctx context.Context, accountID string, role Role) (Session, error)
	Lookup(ctx context.Context, sessionID string) (Session, error)
	Revoke(ctx context.Context, sessionID string) error
	RevokeAccount(ctx context.Context, accountID string) error
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID() (string, error)
}

type TermsStore interface {
	Current(ctx context.Context) (TermsDocument, error)
	PersistAcceptance(ctx context.Context, receipt TermsAcceptanceReceipt) error
	HasAcceptance(ctx context.Context, accountID, version, sha256 string) (bool, error)
}

type AccessMetadataSink interface {
	Record(ctx context.Context, metadata AccessMetadata) error
}

type WireGuardKeyPair struct {
	PrivateKey string
	PublicKey  string
}

type WireGuardKeyGenerator interface {
	Generate(ctx context.Context) (WireGuardKeyPair, error)
}

type WireGuardRemoval struct {
	PeerID  string
	Removed bool
	Error   string
}

type WireGuardController interface {
	RemovePeers(ctx context.Context, peerIDs []string) ([]WireGuardRemoval, error)
	ConfirmPeersRemoved(ctx context.Context, peerIDs []string) (bool, error)
}
