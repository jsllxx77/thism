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
