package internal

import (
	"fmt"

	"github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

// Version is injected by the release build so runtime manifests report the tag.
var Version = "dev"

// StreamProvider implements sdk.PluginProvider and sdk.StepProvider.
type StreamProvider struct{}

// NewStreamProvider creates a new StreamProvider.
func NewStreamProvider() *StreamProvider {
	return &StreamProvider{}
}

// Manifest implements sdk.PluginProvider.
func (p *StreamProvider) Manifest() sdk.PluginManifest {
	return sdk.PluginManifest{
		Name:        "workflow-plugin-stream",
		Version:     sdk.ResolveBuildVersion(Version),
		Author:      "GoCodeAlone",
		Description: "Live video ingest/multiplex/restream (MediaMTX)",
	}
}

// StepTypes implements sdk.StepProvider.
func (p *StreamProvider) StepTypes() []string {
	return []string{"stream.start", "stream.restream"}
}

// CreateStep implements sdk.StepProvider.
func (p *StreamProvider) CreateStep(typeName, name string, config map[string]any) (sdk.StepInstance, error) {
	switch typeName {
	case "stream.start":
		return &StreamStartStep{config: config}, nil
	case "stream.restream":
		return &StreamRestreamStep{config: config}, nil
	}
	return nil, fmt.Errorf("unknown step type: %s", typeName)
}
