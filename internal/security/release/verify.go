// Package release provides Ed25519 signature verification for agent
// self-update binaries. The verifier holds a compile-time-pinned public key;
// the matching private key is held offline by the release signer.
//
// Build-time override: set the public key via ldflags so different
// distributions can pin different keys without code changes:
//
//	go build -ldflags "-X github.com/thism-dev/thism/internal/security/release.PublicKeyBase64=<base64-pubkey>" ./...
//
// When PublicKeyBase64 is empty, the agent treats verification as
// not-configured and refuses to apply any binary update — fail closed.
package release

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// PublicKeyBase64 is the base64-encoded Ed25519 public key used to verify
// agent self-update binaries. Override via ldflags. Empty by default.
var PublicKeyBase64 = ""

var (
	// ErrNoPublicKey is returned when no release public key has been pinned
	// into the binary. The agent fails closed: refuses to self-update.
	ErrNoPublicKey = errors.New("release public key not configured")
	// ErrMissingSignature is returned when a self-update is attempted
	// without a signature.
	ErrMissingSignature = errors.New("release signature missing")
	// ErrInvalidSignature is returned when the provided signature does not
	// verify against the pinned public key.
	ErrInvalidSignature = errors.New("release signature invalid")
)

// Configured reports whether a release public key has been pinned into this
// build. Used by callers that want to surface a clear configuration error.
func Configured() bool {
	return strings.TrimSpace(PublicKeyBase64) != ""
}

// VerifyBinary returns nil iff signatureHex is a valid Ed25519 signature of
// data under the pinned public key. signatureHex is a lowercase or uppercase
// hex string (128 chars = 64 bytes).
//
// Fail-closed semantics:
//   - if no key is pinned, returns ErrNoPublicKey
//   - if signatureHex is empty/whitespace, returns ErrMissingSignature
//   - otherwise returns ErrInvalidSignature on any decoding or
//     verification failure
func VerifyBinary(data []byte, signatureHex string) error {
	pubKeyB64 := strings.TrimSpace(PublicKeyBase64)
	if pubKeyB64 == "" {
		return ErrNoPublicKey
	}
	pubKey, err := decodePublicKey(pubKeyB64)
	if err != nil {
		return fmt.Errorf("%w: decode public key: %v", ErrInvalidSignature, err)
	}
	sigHex := strings.TrimSpace(signatureHex)
	if sigHex == "" {
		return ErrMissingSignature
	}
	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return fmt.Errorf("%w: decode signature hex: %v", ErrInvalidSignature, err)
	}
	if len(sig) != ed25519.SignatureSize {
		return fmt.Errorf("%w: expected %d byte signature, got %d", ErrInvalidSignature, ed25519.SignatureSize, len(sig))
	}
	if !ed25519.Verify(pubKey, data, sig) {
		return ErrInvalidSignature
	}
	return nil
}

func decodePublicKey(b64 string) (ed25519.PublicKey, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, err
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("expected %d byte public key, got %d", ed25519.PublicKeySize, len(raw))
	}
	return ed25519.PublicKey(raw), nil
}
