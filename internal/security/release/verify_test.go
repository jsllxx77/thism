package release

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"testing"
)

func withPinnedKey(t *testing.T, b64 string) {
	t.Helper()
	prev := PublicKeyBase64
	PublicKeyBase64 = b64
	t.Cleanup(func() { PublicKeyBase64 = prev })
}

func generateTestKeypair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return pub, priv
}

func TestVerifyBinary_NoKeyConfigured(t *testing.T) {
	withPinnedKey(t, "")
	err := VerifyBinary([]byte("payload"), "deadbeef")
	if !errors.Is(err, ErrNoPublicKey) {
		t.Fatalf("expected ErrNoPublicKey, got %v", err)
	}
}

func TestVerifyBinary_MissingSignature(t *testing.T) {
	pub, _ := generateTestKeypair(t)
	withPinnedKey(t, base64.StdEncoding.EncodeToString(pub))
	for _, sig := range []string{"", "   ", "\t\n"} {
		if err := VerifyBinary([]byte("payload"), sig); !errors.Is(err, ErrMissingSignature) {
			t.Fatalf("sig=%q expected ErrMissingSignature, got %v", sig, err)
		}
	}
}

func TestVerifyBinary_ValidSignature(t *testing.T) {
	pub, priv := generateTestKeypair(t)
	withPinnedKey(t, base64.StdEncoding.EncodeToString(pub))
	payload := []byte("the quick brown fox")
	sig := hex.EncodeToString(ed25519.Sign(priv, payload))
	if err := VerifyBinary(payload, sig); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestVerifyBinary_TamperedPayload(t *testing.T) {
	pub, priv := generateTestKeypair(t)
	withPinnedKey(t, base64.StdEncoding.EncodeToString(pub))
	sig := hex.EncodeToString(ed25519.Sign(priv, []byte("original")))
	if err := VerifyBinary([]byte("tampered"), sig); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestVerifyBinary_TamperedSignature(t *testing.T) {
	pub, priv := generateTestKeypair(t)
	withPinnedKey(t, base64.StdEncoding.EncodeToString(pub))
	rawSig := ed25519.Sign(priv, []byte("payload"))
	rawSig[0] ^= 0xff
	if err := VerifyBinary([]byte("payload"), hex.EncodeToString(rawSig)); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestVerifyBinary_WrongKey(t *testing.T) {
	pubA, _ := generateTestKeypair(t)
	_, privB := generateTestKeypair(t)
	withPinnedKey(t, base64.StdEncoding.EncodeToString(pubA))
	sig := hex.EncodeToString(ed25519.Sign(privB, []byte("payload")))
	if err := VerifyBinary([]byte("payload"), sig); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestVerifyBinary_BadHex(t *testing.T) {
	pub, _ := generateTestKeypair(t)
	withPinnedKey(t, base64.StdEncoding.EncodeToString(pub))
	if err := VerifyBinary([]byte("payload"), "not-hex-zzz"); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestVerifyBinary_WrongSigLength(t *testing.T) {
	pub, _ := generateTestKeypair(t)
	withPinnedKey(t, base64.StdEncoding.EncodeToString(pub))
	if err := VerifyBinary([]byte("payload"), "deadbeef"); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestVerifyBinary_BadPublicKeyBase64(t *testing.T) {
	withPinnedKey(t, "not-base64-!!!")
	if err := VerifyBinary([]byte("payload"), hex.EncodeToString(make([]byte, ed25519.SignatureSize))); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature (wrapping decode err), got %v", err)
	}
}

func TestConfigured(t *testing.T) {
	withPinnedKey(t, "")
	if Configured() {
		t.Fatal("expected Configured()=false when pubkey empty")
	}
	pub, _ := generateTestKeypair(t)
	withPinnedKey(t, base64.StdEncoding.EncodeToString(pub))
	if !Configured() {
		t.Fatal("expected Configured()=true when pubkey set")
	}
}
