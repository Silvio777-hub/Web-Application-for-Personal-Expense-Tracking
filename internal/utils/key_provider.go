package utils

import (
	"encoding/hex"
	"errors"
	"os"
)

// KeyProvider defines the interface for retrieving cryptographic keys.
// This allows for future integration with Vault, KMS, or HSMs.
type KeyProvider interface {
	GetMasterKey() (string, error)
}

// EnvKeyProvider retrieves keys from environment variables (Standard implementation).
type EnvKeyProvider struct {
	EnvVar string
}

func (p *EnvKeyProvider) GetMasterKey() (string, error) {
	key := os.Getenv(p.EnvVar)
	if key == "" {
		return "", errors.New("master key environment variable not set")
	}
	// Validate hex encoding to prevent runtime crashes
	if _, err := hex.DecodeString(key); err != nil {
		return "", errors.New("invalid hex format for master key")
	}
	return key, nil
}

// StaticKeyProvider is used for testing or pre-loaded configurations.
type StaticKeyProvider struct {
	Key string
}

func (p *StaticKeyProvider) GetMasterKey() (string, error) {
	return p.Key, nil
}
