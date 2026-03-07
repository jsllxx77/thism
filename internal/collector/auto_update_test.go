package collector

import (
  "testing"

  "github.com/thism-dev/thism/internal/models"
)

func TestCollectorAutoUpdateSkipsWhenChecksumMatches(t *testing.T) {
  c := NewWithInterval("http://localhost:12026", "token", "node", "", DefaultReportInterval)
  called := false
  c.selfUpdateFunc = func(cmd models.AgentCommandPayload, report func(models.UpdateJobTargetStatus, string, string) error) error {
    called = true
    return nil
  }

  err := c.maybeApplyRelease(agentReleaseManifest{
    TargetVersion: "abc123456789",
    DownloadURL:   "http://localhost:12026/dl/thism-agent-linux-amd64",
    SHA256:        "same-checksum",
  }, "same-checksum")
  if err != nil {
    t.Fatalf("maybeApplyRelease: %v", err)
  }
  if called {
    t.Fatal("expected self update to be skipped when checksum matches")
  }
}

func TestCollectorAutoUpdateRunsWhenChecksumDiffers(t *testing.T) {
  c := NewWithInterval("http://localhost:12026", "token", "node", "", DefaultReportInterval)
  called := false
  c.selfUpdateFunc = func(cmd models.AgentCommandPayload, report func(models.UpdateJobTargetStatus, string, string) error) error {
    called = true
    if cmd.TargetVersion != "abc123456789" {
      t.Fatalf("expected target version to propagate, got %q", cmd.TargetVersion)
    }
    return nil
  }

  err := c.maybeApplyRelease(agentReleaseManifest{
    TargetVersion: "abc123456789",
    DownloadURL:   "http://localhost:12026/dl/thism-agent-linux-amd64",
    SHA256:        "new-checksum",
  }, "old-checksum")
  if err != nil {
    t.Fatalf("maybeApplyRelease: %v", err)
  }
  if !called {
    t.Fatal("expected self update to run when checksum differs")
  }
}
