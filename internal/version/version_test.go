package version

import "testing"

func TestVersionDefaults(t *testing.T) {
	if Version != "dev" {
		t.Fatalf("expected default Version to be dev, got %q", Version)
	}
	if Commit != "" {
		t.Fatalf("expected default Commit to be empty, got %q", Commit)
	}
	if BuildTime != "" {
		t.Fatalf("expected default BuildTime to be empty, got %q", BuildTime)
	}
}
