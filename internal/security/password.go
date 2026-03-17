package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	passwordHashPrefix      = "argon2id$"
	passwordHashSaltSize    = 16
	passwordHashIterations  = 3
	passwordHashMemoryKB    = 64 * 1024
	passwordHashParallelism = 1
	passwordHashKeyLen      = 32
)

func IsPasswordHash(value string) bool {
	return strings.HasPrefix(value, passwordHashPrefix)
}

func NeedsPasswordHashUpgrade(value string) bool {
	return strings.TrimSpace(value) != "" && !IsPasswordHash(value)
}

func HashPassword(password string) (string, error) {
	salt := make([]byte, passwordHashSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		passwordHashIterations,
		passwordHashMemoryKB,
		passwordHashParallelism,
		passwordHashKeyLen,
	)

	return passwordHashPrefix + hex.EncodeToString(salt) + "$" + hex.EncodeToString(hash), nil
}

func VerifyPassword(password, stored string) bool {
	if stored == "" {
		return false
	}
	if !IsPasswordHash(stored) {
		if len(password) != len(stored) {
			return false
		}
		return subtle.ConstantTimeCompare([]byte(password), []byte(stored)) == 1
	}

	encoded := strings.TrimPrefix(stored, passwordHashPrefix)
	parts := strings.Split(encoded, "$")
	if len(parts) != 2 {
		return false
	}

	salt, err := hex.DecodeString(parts[0])
	if err != nil || len(salt) == 0 {
		return false
	}

	expected, err := hex.DecodeString(parts[1])
	if err != nil || len(expected) == 0 {
		return false
	}

	actual := argon2.IDKey(
		[]byte(password),
		salt,
		passwordHashIterations,
		passwordHashMemoryKB,
		passwordHashParallelism,
		uint32(len(expected)),
	)

	return subtle.ConstantTimeCompare(actual, expected) == 1
}
