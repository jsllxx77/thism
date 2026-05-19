package collector

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/thism-dev/thism/internal/models"
	"github.com/thism-dev/thism/internal/security/release"
)

func newSignatureTestCollector(t *testing.T, serverURL string) *Collector {
	t.Helper()
	c := NewWithInterval(serverURL, "token", "node", "", DefaultReportInterval)
	c.agentVersion = "1.0.0"
	return c
}

func serveBinary(t *testing.T, payload []byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
}

func withReleasePinnedKey(t *testing.T, b64 string) {
	t.Helper()
	prev := release.PublicKeyBase64
	release.PublicKeyBase64 = b64
	t.Cleanup(func() { release.PublicKeyBase64 = prev })
}

// noopReport is a status reporter that swallows every status update —
// keeps tests focused on the verification logic rather than the
// reporting plumbing (which is exercised elsewhere).
func noopReport(models.UpdateJobTargetStatus, string, string) error { return nil }

func TestRunSelfUpdate_RejectsMissingSignatureWhenKeyPinned(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	withReleasePinnedKey(t, base64.StdEncoding.EncodeToString(pub))

	payload := []byte("agent-bin")
	srv := serveBinary(t, payload)
	defer srv.Close()

	c := newSignatureTestCollector(t, srv.URL)
	digest := sha256.Sum256(payload)

	err = c.runSelfUpdate(models.AgentCommandPayload{
		Kind:          models.AgentCommandKindSelfUpdate,
		TargetVersion: "v2",
		DownloadURL:   srv.URL,
		SHA256:        hex.EncodeToString(digest[:]),
		Signature:     "",
	}, noopReport)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, release.ErrMissingSignature) {
		t.Fatalf("expected ErrMissingSignature, got %v", err)
	}
}

func TestRunSelfUpdate_RejectsInvalidSignature(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	withReleasePinnedKey(t, base64.StdEncoding.EncodeToString(pub))

	payload := []byte("agent-bin")
	srv := serveBinary(t, payload)
	defer srv.Close()

	c := newSignatureTestCollector(t, srv.URL)
	digest := sha256.Sum256(payload)

	// Signature is well-formed (64 bytes hex) but won't verify under pub.
	fakeSig := make([]byte, ed25519.SignatureSize)
	fakeSig[0] = 0x01

	err = c.runSelfUpdate(models.AgentCommandPayload{
		Kind:          models.AgentCommandKindSelfUpdate,
		TargetVersion: "v2",
		DownloadURL:   srv.URL,
		SHA256:        hex.EncodeToString(digest[:]),
		Signature:     hex.EncodeToString(fakeSig),
	}, noopReport)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, release.ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestRunSelfUpdate_RejectsWhenNoKeyPinned(t *testing.T) {
	withReleasePinnedKey(t, "")

	payload := []byte("agent-bin")
	srv := serveBinary(t, payload)
	defer srv.Close()

	c := newSignatureTestCollector(t, srv.URL)
	digest := sha256.Sum256(payload)

	err := c.runSelfUpdate(models.AgentCommandPayload{
		Kind:          models.AgentCommandKindSelfUpdate,
		TargetVersion: "v2",
		DownloadURL:   srv.URL,
		SHA256:        hex.EncodeToString(digest[:]),
		Signature:     "deadbeef",
	}, noopReport)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, release.ErrNoPublicKey) {
		t.Fatalf("expected ErrNoPublicKey, got %v", err)
	}
}

func TestRunSelfUpdate_FailsOnSHA256MismatchBeforeSignature(t *testing.T) {
	// Sanity: SHA256 mismatch still wins (we don't reach signature check),
	// so attackers can't waste server CPU on signature verification of
	// payloads that don't even match the declared hash.
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	withReleasePinnedKey(t, base64.StdEncoding.EncodeToString(pub))

	srv := serveBinary(t, []byte("actual-payload"))
	defer srv.Close()

	c := newSignatureTestCollector(t, srv.URL)
	err = c.runSelfUpdate(models.AgentCommandPayload{
		Kind:          models.AgentCommandKindSelfUpdate,
		TargetVersion: "v2",
		DownloadURL:   srv.URL,
		SHA256:        strings.Repeat("00", 32),
		Signature:     "deadbeef",
	}, noopReport)
	if err == nil || !strings.Contains(err.Error(), "sha256 mismatch") {
		t.Fatalf("expected sha256 mismatch, got %v", err)
	}
}
