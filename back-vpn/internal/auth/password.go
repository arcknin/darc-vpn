package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type Argon2idParams struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

func DefaultParams() Argon2idParams {
	return Argon2idParams{Memory: 64 * 1024, Iterations: 2, Parallelism: 1, SaltLength: 16, KeyLength: 32}
}

// TestParams keeps the same Argon2id behavior with a smaller memory cost for local tests.
func TestParams() Argon2idParams {
	return Argon2idParams{Memory: 8 * 1024, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}
}

type Argon2idHasher struct {
	params Argon2idParams
}

func NewArgon2idHasher(params Argon2idParams) *Argon2idHasher {
	return &Argon2idHasher{params: params}
}

func (h *Argon2idHasher) Hash(ctx context.Context, plaintext string) (string, error) {
	if err := validateParams(h.params); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	salt := make([]byte, h.params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	key := argon2.IDKey([]byte(plaintext), salt, h.params.Iterations, h.params.Memory, h.params.Parallelism, h.params.KeyLength)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", h.params.Memory, h.params.Iterations, h.params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

func (h *Argon2idHasher) Verify(ctx context.Context, encoded, plaintext string) (bool, error) {
	params, salt, expected, err := decodeHash(encoded)
	if err != nil {
		return false, err
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	actual := argon2.IDKey([]byte(plaintext), salt, params.Iterations, params.Memory, params.Parallelism, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}

func decodeHash(encoded string) (Argon2idParams, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return Argon2idParams{}, nil, nil, errors.New("invalid password hash")
	}
	var params Argon2idParams
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.Memory, &params.Iterations, &params.Parallelism); err != nil {
		return Argon2idParams{}, nil, nil, errors.New("invalid password hash")
	}
	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil {
		return Argon2idParams{}, nil, nil, errors.New("invalid password hash")
	}
	expected, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil {
		return Argon2idParams{}, nil, nil, errors.New("invalid password hash")
	}
	params.SaltLength = uint32(len(salt))
	params.KeyLength = uint32(len(expected))
	if err := validateParams(params); err != nil {
		return Argon2idParams{}, nil, nil, errors.New("invalid password hash")
	}
	return params, salt, expected, nil
}

func validateParams(params Argon2idParams) error {
	if params.Memory < 8*1024 || params.Memory > 1024*1024 || params.Iterations < 1 || params.Iterations > 10 ||
		params.Parallelism < 1 || params.Parallelism > 16 || params.SaltLength < 16 || params.SaltLength > 64 ||
		params.KeyLength < 16 || params.KeyLength > 64 {
		return errors.New("invalid Argon2id parameters")
	}
	return nil
}
