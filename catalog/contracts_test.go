package catalog

import (
	"slices"
	"testing"

	core "github.com/GoCodeAlone/workflow-plugin-compute-core/protocol"
)

func TestPublicCatalogExportsVideoStreamContracts(t *testing.T) {
	providers := ProviderContracts()
	if len(providers) != 1 {
		t.Fatalf("provider contract count = %d", len(providers))
	}
	provider := providers[0]
	if provider.PluginID != "workflow-plugin-stream" || provider.ProviderID != "video-stream" {
		t.Fatalf("provider tuple = %q/%q", provider.PluginID, provider.ProviderID)
	}
	if !slices.Contains(provider.WorkloadKinds, string(core.WorkloadVideoStream)) || !provider.SupportsOperation("start_stream") {
		t.Fatalf("provider contract does not advertise video stream start: %+v", provider)
	}
	if err := provider.Validate(); err != nil {
		t.Fatalf("provider contract invalid: %v", err)
	}

	adapters := RuntimeAdapterContracts()
	if len(adapters) != 1 {
		t.Fatalf("runtime adapter count = %d", len(adapters))
	}
	adapter := adapters[0]
	if adapter.AdapterID != "stream-service-session" || !adapter.SupportsAdapterKind(core.RuntimeAdapterServiceSession) || !adapter.Supports(core.WorkloadVideoStream) {
		t.Fatalf("runtime adapter contract does not advertise stream service-session: %+v", adapter)
	}
	if err := adapter.Validate(); err != nil {
		t.Fatalf("runtime adapter contract invalid: %v", err)
	}
}
