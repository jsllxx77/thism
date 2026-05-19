package main

import (
	"testing"

	sharedversion "github.com/thism-dev/thism/internal/version"
)

func TestRuntimeAgentVersionUsesSharedVersion(t *testing.T) {
	original := sharedversion.Version
	sharedversion.Version = "v9.9.9-test"
	t.Cleanup(func() {
		sharedversion.Version = original
	})

	if got := runtimeAgentVersion(); got != "v9.9.9-test" {
		t.Fatalf("expected runtime agent version to come from shared version package, got %q", got)
	}
}

func TestEnvOrReturnsEnvWhenSet(t *testing.T) {
	t.Setenv("THISM_AGENT_TEST_KEY", "from-env")
	if got := envOr("THISM_AGENT_TEST_KEY", "fallback"); got != "from-env" {
		t.Fatalf("expected env value to win over fallback, got %q", got)
	}
}

func TestEnvOrReturnsFallbackWhenUnset(t *testing.T) {
	if got := envOr("THISM_AGENT_TEST_KEY_UNSET", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback when env unset, got %q", got)
	}
}
