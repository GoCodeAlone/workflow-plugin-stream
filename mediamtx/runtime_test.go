package mediamtx

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	core "github.com/GoCodeAlone/workflow-plugin-compute-core/protocol"
	"gopkg.in/yaml.v3"
)

func TestRenderConfigEnablesRequestedProtocolsAndAuthHook(t *testing.T) {
	t.Parallel()

	spec := core.StreamSpec{
		IngestProtocols: []string{"rtmp", "srt", "whip"},
		ViewerEgress: core.ViewerEgressConfig{
			HLS:  true,
			WHEP: true,
		},
		Recording: true,
	}

	rendered, err := RenderConfig(spec, ConfigOptions{
		PathName:        "orgs/acme/pools/video/live/main",
		AuthHTTPAddress: "http://127.0.0.1:18080/stream/auth",
	})
	if err != nil {
		t.Fatalf("RenderConfig: %v", err)
	}

	var cfg map[string]any
	if err := yaml.Unmarshal(rendered, &cfg); err != nil {
		t.Fatalf("parse rendered yaml: %v\n%s", err, rendered)
	}

	assertBool(t, cfg, "rtmp", true)
	assertBool(t, cfg, "srt", true)
	assertBool(t, cfg, "webrtc", true)
	assertBool(t, cfg, "hls", true)
	assertBool(t, cfg, "rtsp", false)
	assertBool(t, cfg, "api", true)
	assertString(t, cfg, "authMethod", "http")
	assertString(t, cfg, "authHTTPAddress", "http://127.0.0.1:18080/stream/auth")

	paths, ok := cfg["paths"].(map[string]any)
	if !ok {
		t.Fatalf("paths missing or wrong type: %#v", cfg["paths"])
	}
	path, ok := paths["orgs/acme/pools/video/live/main"].(map[string]any)
	if !ok {
		t.Fatalf("configured stream path missing: %#v", paths)
	}
	assertStringInMap(t, path, "source", "publisher")
	assertBoolInMap(t, path, "record", true)
}

func TestRenderConfigDisablesUnrequestedProtocols(t *testing.T) {
	t.Parallel()

	rendered, err := RenderConfig(core.StreamSpec{
		IngestProtocols: []string{"rtmp"},
		ViewerEgress:    core.ViewerEgressConfig{HLS: true},
	}, ConfigOptions{
		PathName:        "live/rtmp-only",
		AuthHTTPAddress: "http://127.0.0.1:18080/stream/auth",
	})
	if err != nil {
		t.Fatalf("RenderConfig: %v", err)
	}

	var cfg map[string]any
	if err := yaml.Unmarshal(rendered, &cfg); err != nil {
		t.Fatalf("parse rendered yaml: %v", err)
	}

	assertBool(t, cfg, "rtmp", true)
	assertBool(t, cfg, "srt", false)
	assertBool(t, cfg, "webrtc", false)
	assertBool(t, cfg, "hls", true)
}

func TestManagedRuntimeBundlesPinMediaMTXAndValidate(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("../managed-runtime-bundles.json")
	if err != nil {
		t.Fatalf("read managed-runtime-bundles.json: %v", err)
	}
	var manifest struct {
		Bundles []core.ManagedRuntimeBundleDescriptor `json:"bundles"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("decode managed runtime bundles: %v", err)
	}
	if len(manifest.Bundles) != 1 {
		t.Fatalf("bundle count = %d, want 1", len(manifest.Bundles))
	}
	bundle := manifest.Bundles[0]
	if bundle.Family != core.RuntimeBackendFamilyMediaMTX {
		t.Fatalf("family = %q", bundle.Family)
	}
	if bundle.Version != "v1.19.1" {
		t.Fatalf("version = %q", bundle.Version)
	}
	if bundle.ArtifactDigest != "sha256:035ee04f91b1c7a0c02e13b2139ca2456e43b6bd6a80e3100e8c228556e07807" {
		t.Fatalf("artifact digest = %q", bundle.ArtifactDigest)
	}
	if bundle.SignatureName != "github_artifact_attestation_bundle" {
		t.Fatalf("signature name = %q", bundle.SignatureName)
	}
	if bundle.SignatureDigest != "sha256:a008c9a89b4040a3f6903df99616dcf46c68ef619ee6ef204e10c60455eccf6f" {
		t.Fatalf("signature digest = %q", bundle.SignatureDigest)
	}
	if err := bundle.ValidateAt(time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("bundle descriptor did not validate: %v", err)
	}
}

func assertBool(t *testing.T, cfg map[string]any, key string, want bool) {
	t.Helper()
	got, ok := cfg[key].(bool)
	if !ok || got != want {
		t.Fatalf("%s = %#v, want %v", key, cfg[key], want)
	}
}

func assertString(t *testing.T, cfg map[string]any, key string, want string) {
	t.Helper()
	got, ok := cfg[key].(string)
	if !ok || got != want {
		t.Fatalf("%s = %#v, want %q", key, cfg[key], want)
	}
}

func assertBoolInMap(t *testing.T, cfg map[string]any, key string, want bool) {
	t.Helper()
	got, ok := cfg[key].(bool)
	if !ok || got != want {
		t.Fatalf("%s = %#v, want %v", key, cfg[key], want)
	}
}

func assertStringInMap(t *testing.T, cfg map[string]any, key string, want string) {
	t.Helper()
	got, ok := cfg[key].(string)
	if !ok || got != want {
		t.Fatalf("%s = %#v, want %q", key, cfg[key], want)
	}
}
