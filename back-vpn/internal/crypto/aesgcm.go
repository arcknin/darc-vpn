package crypto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	cryptorand "crypto/rand"
	"encoding/binary"
	"io"

	"darc-vpn/back-vpn/internal/domain"
)

const Version = "aes-256-gcm-v1"

type AESGCM struct {
	aead cipher.AEAD
	rand io.Reader
}

func NewAESGCM(masterKey []byte, random io.Reader) (*AESGCM, error) {
	if len(masterKey) != 32 {
		return nil, recoveryError()
	}
	block, err := aes.NewCipher(append([]byte(nil), masterKey...))
	if err != nil {
		return nil, recoveryError()
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, recoveryError()
	}
	if random == nil {
		random = cryptorand.Reader
	}
	return &AESGCM{aead: aead, rand: random}, nil
}

func (c *AESGCM) Encrypt(ctx context.Context, plaintext, associatedData []byte) (domain.EncryptedPrivateProfile, error) {
	if c == nil || c.aead == nil || ctx.Err() != nil {
		return domain.EncryptedPrivateProfile{}, recoveryError()
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(c.rand, nonce); err != nil {
		return domain.EncryptedPrivateProfile{}, recoveryError()
	}
	return domain.EncryptedPrivateProfile{
		Ciphertext: c.aead.Seal(nil, nonce, plaintext, associatedData),
		Nonce:      nonce,
		Version:    Version,
	}, nil
}

func (c *AESGCM) Decrypt(ctx context.Context, encrypted domain.EncryptedPrivateProfile, associatedData []byte) ([]byte, error) {
	if c == nil || c.aead == nil || ctx.Err() != nil || encrypted.Version != Version || len(encrypted.Nonce) != c.aead.NonceSize() {
		return nil, recoveryError()
	}
	plaintext, err := c.aead.Open(nil, encrypted.Nonce, encrypted.Ciphertext, associatedData)
	if err != nil {
		return nil, recoveryError()
	}
	return plaintext, nil
}

func AssociatedData(accountID, profileID, version string) []byte {
	values := []string{accountID, profileID, version}
	result := make([]byte, 0, len(accountID)+len(profileID)+len(version)+12)
	for _, value := range values {
		var size [4]byte
		binary.BigEndian.PutUint32(size[:], uint32(len(value)))
		result = append(result, size[:]...)
		result = append(result, value...)
	}
	return result
}

func recoveryError() error {
	return domain.NewCodedError(domain.ErrProfileRecoveryFailed, "private profile could not be recovered")
}
