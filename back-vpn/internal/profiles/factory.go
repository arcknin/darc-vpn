package profiles

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"strings"

	profilecrypto "darc-vpn/back-vpn/internal/crypto"
	"darc-vpn/back-vpn/internal/domain"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type AddressResolver func(accountID, profileID string) string

type FactoryConfig struct {
	IDs        domain.IDGenerator
	Clock      domain.Clock
	Keys       domain.WireGuardKeyGenerator
	Cipher     domain.ProfileCipher
	Address    AddressResolver
	AllowedIPs []string
	DNS        []string
}

type Factory struct{ config FactoryConfig }

func NewFactory(config FactoryConfig) *Factory { return &Factory{config: config} }

func (f *Factory) Create(ctx context.Context, accountID, displayName string, primary bool) (domain.ProfileRecord, error) {
	if accountID == "" || strings.TrimSpace(displayName) == "" || f == nil || f.config.IDs == nil || f.config.Clock == nil ||
		f.config.Keys == nil || f.config.Cipher == nil || f.config.Address == nil {
		return domain.ProfileRecord{}, domain.NewCodedError(domain.ErrInvalidRequest, "invalid profile")
	}
	profileID, err := f.config.IDs.NewID()
	if err != nil {
		return domain.ProfileRecord{}, err
	}
	keys, err := f.config.Keys.Generate(ctx)
	if err != nil {
		return domain.ProfileRecord{}, err
	}
	payload, err := json.Marshal(privatePayload{PrivateKey: keys.PrivateKey})
	if err != nil {
		return domain.ProfileRecord{}, err
	}
	aad := profilecrypto.AssociatedData(accountID, profileID, profilecrypto.Version)
	encrypted, err := f.config.Cipher.Encrypt(ctx, payload, aad)
	if err != nil {
		return domain.ProfileRecord{}, err
	}
	return domain.ProfileRecord{
		ID:               profileID,
		AccountID:        accountID,
		DisplayName:      strings.TrimSpace(displayName),
		IsPrimary:        primary,
		Endpoint:         domain.VPNEndpoint,
		PeerPublicKey:    keys.PublicKey,
		Address:          f.config.Address(accountID, profileID),
		AllowedIPs:       append([]string(nil), f.config.AllowedIPs...),
		DNS:              append([]string(nil), f.config.DNS...),
		EncryptedPrivate: encrypted,
		CreatedAt:        f.config.Clock.Now().UTC(),
	}, nil
}

type privatePayload struct {
	PrivateKey   string `json:"privateKey"`
	PresharedKey string `json:"presharedKey,omitempty"`
}

type WireGuardKeyGenerator struct{ random io.Reader }

func NewWireGuardKeyGenerator(random io.Reader) *WireGuardKeyGenerator {
	if random == nil {
		random = rand.Reader
	}
	return &WireGuardKeyGenerator{random: random}
}

func (g *WireGuardKeyGenerator) Generate(ctx context.Context) (domain.WireGuardKeyPair, error) {
	if g == nil || g.random == nil || ctx.Err() != nil {
		return domain.WireGuardKeyPair{}, errors.New("WireGuard key generation unavailable")
	}
	var bytes [wgtypes.KeyLen]byte
	if _, err := io.ReadFull(g.random, bytes[:]); err != nil {
		return domain.WireGuardKeyPair{}, err
	}
	bytes[0] &= 248
	bytes[31] &= 127
	bytes[31] |= 64
	private := wgtypes.Key(bytes)
	return domain.WireGuardKeyPair{PrivateKey: private.String(), PublicKey: private.PublicKey().String()}, nil
}
