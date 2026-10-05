package domain

import "time"

type Role string

const (
	RoleFriend Role = "friend"
	RoleAdmin  Role = "admin"
)

type AccountStatus string

const (
	AccountActive  AccountStatus = "active"
	AccountBlocked AccountStatus = "blocked"
)

type AccountRecord struct {
	ID           string
	Username     string
	Role         Role
	Status       AccountStatus
	CreatedAt    time.Time
	PrimaryID    string
	ProfileCount int
}

// ProfileRecord contains only encrypted private material. Plaintext keys are
// intentionally absent from the persistence boundary.
type ProfileRecord struct {
	ProvisioningState string
	ProvisioningError string
	ID                string
	AccountID         string
	DisplayName       string
	IsPrimary         bool
	Endpoint          string
	PeerPublicKey     string
	Address           string
	AllowedIPs        []string
	DNS               []string
	EncryptedPrivate  EncryptedPrivateProfile
	CreatedAt         time.Time
}

type EncryptedPrivateProfile struct {
	Ciphertext []byte
	Nonce      []byte
	Version    string
}

// ProfileSummary is safe for an authenticated friend to list or display.
// It has no private key, preshared key, or exportable configuration field.
type ProfileSummary struct {
	ID            string   `json:"id"`
	DisplayName   string   `json:"displayName"`
	IsPrimary     bool     `json:"isPrimary"`
	Endpoint      string   `json:"endpoint"`
	PeerPublicKey string   `json:"peerPublicKey"`
	Address       string   `json:"address"`
	AllowedIPs    []string `json:"allowedIPs"`
	DNS           []string `json:"dns"`
}

// ProfilePrivate is delivered only to the authenticated owner after the
// terms gate has succeeded. Its fields are never part of admin DTOs.
type ProfilePrivate struct {
	ServerPublicKey string   `json:"serverPublicKey,omitempty"`
	EndpointPort    int      `json:"endpointPort,omitempty"`
	ID              string   `json:"id"`
	DisplayName     string   `json:"displayName"`
	Endpoint        string   `json:"endpoint"`
	Address         string   `json:"address"`
	AllowedIPs      []string `json:"allowedIPs"`
	DNS             []string `json:"dns"`
	PeerPublicKey   string   `json:"peerPublicKey"`
	PrivateKey      string   `json:"privateKey"`
	PresharedKey    string   `json:"presharedKey,omitempty"`
}

// AdminProfileMetadata deliberately excludes all secret and exportable
// configuration material.
type AdminProfileMetadata struct {
	ProvisioningState string    `json:"provisioningState,omitempty"`
	ProvisioningError string    `json:"provisioningError,omitempty"`
	ID                string    `json:"id"`
	AccountID         string    `json:"accountId"`
	DisplayName       string    `json:"displayName"`
	IsPrimary         bool      `json:"isPrimary"`
	PeerPublicKey     string    `json:"peerPublicKey"`
	CreatedAt         time.Time `json:"createdAt"`
}

type AdminAccountMetadata struct {
	ID           string                 `json:"id"`
	Username     string                 `json:"username"`
	Status       AccountStatus          `json:"status"`
	CreatedAt    time.Time              `json:"createdAt"`
	PrimaryID    string                 `json:"primaryProfileId"`
	ProfileCount int                    `json:"profileCount"`
	Profiles     []AdminProfileMetadata `json:"profiles"`
}

type TermsDocument struct {
	AcceptanceRequired bool   `json:"acceptanceRequired"`
	Version            string `json:"version"`
	Text               string `json:"text"`
	SHA256             string `json:"sha256"`
}

type TermsAcceptanceReceipt struct {
	AccountID  string    `json:"accountId"`
	Version    string    `json:"version"`
	SHA256     string    `json:"sha256"`
	Accepted   bool      `json:"accepted"`
	AcceptedAt time.Time `json:"acceptedAt"`
	EventID    string    `json:"eventId"`
}

type RevocationStatus string

const (
	RevocationPending   RevocationStatus = "pending"
	RevocationFailed    RevocationStatus = "failed"
	RevocationCompleted RevocationStatus = "completed"
)

type PeerRevocation struct {
	ProfileID string `json:"profileId"`
	PeerID    string `json:"peerId"`
	Removed   bool   `json:"removed"`
	Error     string `json:"error,omitempty"`
}

type RevocationOperation struct {
	AccountID string           `json:"accountId"`
	Status    RevocationStatus `json:"status"`
	Peers     []PeerRevocation `json:"peers"`
	Error     string           `json:"error,omitempty"`
	UpdatedAt time.Time        `json:"updatedAt"`
}

type Session struct {
	ID        string
	AccountID string
	Role      Role
	ExpiresAt time.Time
}

type AccessMetadata struct {
	EventID   string
	EventType string
	AccountID string
	ProfileID string
	SourceIP  string
	At        time.Time
}

func (p ProfileRecord) Summary() (ProfileSummary, error) {
	if err := ValidateVPNEndpoint(p.Endpoint); err != nil {
		return ProfileSummary{}, err
	}
	return ProfileSummary{
		ID:            p.ID,
		DisplayName:   p.DisplayName,
		IsPrimary:     p.IsPrimary,
		Endpoint:      p.Endpoint,
		PeerPublicKey: p.PeerPublicKey,
		Address:       p.Address,
		AllowedIPs:    append([]string(nil), p.AllowedIPs...),
		DNS:           append([]string(nil), p.DNS...),
	}, nil
}

func (p ProfileRecord) AdminMetadata() (AdminProfileMetadata, error) {
	if err := ValidateVPNEndpoint(p.Endpoint); err != nil {
		return AdminProfileMetadata{
			ProvisioningState: p.ProvisioningState, ProvisioningError: p.ProvisioningError}, err
	}
	return AdminProfileMetadata{
		ProvisioningState: p.ProvisioningState, ProvisioningError: p.ProvisioningError,
		ID:            p.ID,
		AccountID:     p.AccountID,
		DisplayName:   p.DisplayName,
		IsPrimary:     p.IsPrimary,
		PeerPublicKey: p.PeerPublicKey,
		CreatedAt:     p.CreatedAt,
	}, nil
}

// ManagedPeer is the durable desired state, never plaintext private material.
type ManagedPeer struct {
	ProfileID string
	AccountID string
	PublicKey string
	Address   string
	State     string
	Error     string
	Revision  int64
	UpdatedAt time.Time
	Active    bool
}

// LegacyPeer is an operator-only manifest/runtime record; never an API DTO.
type LegacyPeer struct {
	PublicKey    string   `json:"publicKey"`
	AllowedIPs   []string `json:"allowedIPs"`
	PresharedKey string   `json:"presharedKey,omitempty"`
	Keepalive    int      `json:"keepalive"`
}
