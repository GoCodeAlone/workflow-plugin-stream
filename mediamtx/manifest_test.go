package mediamtx

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	core "github.com/GoCodeAlone/workflow-plugin-compute-core/protocol"
)

func TestBuildStreamManifestHashesSegmentsAndReflectsNonce(t *testing.T) {
	tmp := t.TempDir()
	seg0 := filepath.Join(tmp, "seg-0.ts")
	seg1 := filepath.Join(tmp, "seg-1.ts")
	if err := os.WriteFile(seg0, []byte("segment-zero-nonce-abc"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(seg1, []byte("segment-one"), 0o600); err != nil {
		t.Fatal(err)
	}
	started := time.Unix(1000, 0).UTC()
	issued := core.LivenessNonce{Nonce: "nonce-abc", IssuedAt: started.Add(500 * time.Millisecond)}

	manifest, err := BuildStreamManifest(ManifestInput{
		StreamID:        "stream-1",
		StartedAt:       started,
		SegmentDuration: time.Second,
		SegmentPaths:    []string{seg0, seg1},
		IssuedNonces:    []core.LivenessNonce{issued},
		DeliveryReceipts: []core.DeliveryReceipt{{
			ConsumerID: "viewer-1",
			Kind:       core.ConsumerViewer,
			SeqStart:   0,
			SeqEnd:     0,
			Bytes:      int64(len("segment-zero-nonce-abc")),
		}},
	})
	if err != nil {
		t.Fatalf("BuildStreamManifest: %v", err)
	}
	if len(manifest.Segments) != 2 {
		t.Fatalf("segment count = %d", len(manifest.Segments))
	}
	if manifest.Segments[0].SHA256 == "" || manifest.Segments[0].Provenance != core.ProvenanceHostVerified {
		t.Fatalf("segment[0] not host-verified with hash: %+v", manifest.Segments[0])
	}
	if len(manifest.LivenessNonces) != 1 || manifest.LivenessNonces[0].ReflectedInSeq != 0 {
		t.Fatalf("nonce reflection = %+v", manifest.LivenessNonces)
	}
	if manifest.DeliveryReceipts[0].SHA256 != manifest.Segments[0].SHA256 {
		t.Fatalf("receipt hash = %q, segment hash = %q", manifest.DeliveryReceipts[0].SHA256, manifest.Segments[0].SHA256)
	}
	if err := core.VerifyManifest(manifest, []core.LivenessNonce{issued}, 2*time.Second); err != nil {
		t.Fatalf("manifest did not verify: %v", err)
	}
}
