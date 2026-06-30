package internal

import (
	"encoding/json"
	"os"
	"slices"
	"testing"

	core "github.com/GoCodeAlone/workflow-plugin-compute-core/protocol"
)

func TestPluginManifestAdvertisesStreamScaffold(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("../plugin.json")
	if err != nil {
		t.Fatalf("read plugin.json: %v", err)
	}
	var manifest struct {
		Name               string `json:"name"`
		ProvidersRef       string `json:"providersRef"`
		RuntimeAdaptersRef string `json:"runtimeAdaptersRef"`
		MinEngineVersion   string `json:"minEngineVersion"`
		Capabilities       struct {
			StepTypes []string `json:"stepTypes"`
		} `json:"capabilities"`
		Dependencies []struct {
			Name       string `json:"name"`
			Constraint string `json:"constraint"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("decode plugin.json: %v", err)
	}
	if manifest.Name != "workflow-plugin-stream" {
		t.Fatalf("name = %q", manifest.Name)
	}
	if manifest.ProvidersRef != "stream-providers.json" {
		t.Fatalf("providersRef = %q", manifest.ProvidersRef)
	}
	if manifest.RuntimeAdaptersRef != "runtime-adapters.json" {
		t.Fatalf("runtimeAdaptersRef = %q", manifest.RuntimeAdaptersRef)
	}
	if manifest.MinEngineVersion == "" {
		t.Fatalf("minEngineVersion is required")
	}
	for _, want := range []string{"stream.start", "stream.restream"} {
		if !slices.Contains(manifest.Capabilities.StepTypes, want) {
			t.Fatalf("stepTypes missing %q: %#v", want, manifest.Capabilities.StepTypes)
		}
	}
	if !slices.ContainsFunc(manifest.Dependencies, func(dep struct {
		Name       string `json:"name"`
		Constraint string `json:"constraint"`
	}) bool {
		return dep.Name == "workflow-plugin-compute-core" && dep.Constraint == ">=0.8.0"
	}) {
		t.Fatalf("dependencies missing workflow-plugin-compute-core >=0.8.0: %#v", manifest.Dependencies)
	}
	if core.WorkloadVideoStream != "video-stream" {
		t.Fatalf("compute-core video-stream workload kind not available")
	}
}

func TestRuntimeStepTypesMatchManifest(t *testing.T) {
	t.Parallel()
	plugin := NewStreamProvider()
	for _, want := range []string{"stream.start", "stream.restream"} {
		if !slices.Contains(plugin.StepTypes(), want) {
			t.Fatalf("runtime step types missing %q: %#v", want, plugin.StepTypes())
		}
	}
}
