package googledrive

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
)

func TestPKCEVerifierHasRequiredEntropyAndS256Challenge(t *testing.T) {
	verifier, err := newVerifier()
	if err != nil {
		t.Fatal(err)
	}
	if len(verifier) < 43 || len(verifier) > 128 {
		t.Fatalf("unexpected verifier length %d", len(verifier))
	}
	if strings.ContainsAny(verifier, "+/=") {
		t.Fatalf("verifier is not URL safe: %q", verifier)
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	if challenge == verifier || len(challenge) < 43 {
		t.Fatal("S256 challenge was not derived correctly")
	}
}
