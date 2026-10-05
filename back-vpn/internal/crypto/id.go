package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

type OpaqueIDGenerator struct {
	random io.Reader
}

func NewOpaqueIDGenerator(random io.Reader) *OpaqueIDGenerator {
	if random == nil {
		random = rand.Reader
	}
	return &OpaqueIDGenerator{random: random}
}

func (g *OpaqueIDGenerator) NewID() (string, error) {
	if g == nil || g.random == nil {
		return "", errors.New("opaque ID generation unavailable")
	}
	value := make([]byte, 16)
	if _, err := io.ReadFull(g.random, value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
