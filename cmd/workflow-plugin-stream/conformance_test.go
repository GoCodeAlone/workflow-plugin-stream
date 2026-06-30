package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunConformanceRequiresRealArtifactForEachIngest(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	artifact := filepath.Join(dir, "artifact.json")
	if err := os.WriteFile(artifact, []byte(`{"status":"ok","segments":1}`), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	for _, ingest := range []string{"rtmp", "srt", "whip"} {
		t.Run(ingest, func(t *testing.T) {
			t.Parallel()
			if err := runConformance([]string{"--ingest", ingest, "--egress", "hls", "--artifact", artifact}); err != nil {
				t.Fatalf("runConformance(%s): %v", ingest, err)
			}
		})
	}
}

func TestRunConformanceRejectsMissingArtifact(t *testing.T) {
	t.Parallel()

	err := runConformance([]string{"--ingest", "rtmp", "--egress", "hls", "--artifact", filepath.Join(t.TempDir(), "missing.json")})
	if err == nil {
		t.Fatal("runConformance succeeded with missing artifact")
	}
}
