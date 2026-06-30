package internal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	core "github.com/GoCodeAlone/workflow-plugin-compute-core/protocol"
)

func TestProviderContractsAdvertiseVideoStreamOperations(t *testing.T) {
	t.Parallel()

	contracts := ProviderContracts()
	if len(contracts) != 1 {
		t.Fatalf("contract count = %d, want 1", len(contracts))
	}
	contract := contracts[0]
	if err := contract.Validate(); err != nil {
		t.Fatalf("provider contract invalid: %v", err)
	}
	if !slices.Contains(contract.WorkloadKinds, string(core.WorkloadVideoStream)) {
		t.Fatalf("workload kinds = %#v", contract.WorkloadKinds)
	}
	for _, want := range []string{
		"start_stream",
		"set_transform",
		"add_destination",
		"remove_destination",
		"add_rendition",
		"stop_stream",
	} {
		if !contract.SupportsOperation(want) {
			t.Fatalf("operation %q missing: %#v", want, contract.Operations)
		}
	}
	assertContractSchemaDigests(t, contract)
}

func TestStreamProviderAndRuntimeAdapterManifestsValidate(t *testing.T) {
	t.Parallel()

	var providers struct {
		Version  string                  `json:"version"`
		Provider []core.ProviderContract `json:"providers"`
	}
	readJSON(t, filepath.Join("..", "stream-providers.json"), &providers)
	if providers.Version != "stream-provider-contracts.v1" {
		t.Fatalf("provider catalog version = %q", providers.Version)
	}
	if len(providers.Provider) != 1 {
		t.Fatalf("provider manifest count = %d", len(providers.Provider))
	}
	if err := providers.Provider[0].Validate(); err != nil {
		t.Fatalf("provider manifest invalid: %v", err)
	}
	assertContractSchemaDigests(t, providers.Provider[0])

	var adapters struct {
		Version         string                        `json:"version"`
		ProtocolVersion string                        `json:"protocol_version"`
		Adapters        []core.RuntimeAdapterContract `json:"adapters"`
	}
	readJSON(t, filepath.Join("..", "runtime-adapters.json"), &adapters)
	if adapters.Version != "1" || adapters.ProtocolVersion != core.Version {
		t.Fatalf("runtime adapter catalog header = %+v", adapters)
	}
	if len(adapters.Adapters) != 1 {
		t.Fatalf("adapter count = %d", len(adapters.Adapters))
	}
	if err := adapters.Adapters[0].Validate(); err != nil {
		t.Fatalf("runtime adapter contract invalid: %v", err)
	}
	if !adapters.Adapters[0].SupportsAdapterKind(core.RuntimeAdapterServiceSession) ||
		!adapters.Adapters[0].Supports(core.WorkloadVideoStream) {
		t.Fatalf("adapter does not support service-session video-stream: %#v", adapters.Adapters[0])
	}
}

func assertContractSchemaDigests(t *testing.T, contract core.ProviderContract) {
	t.Helper()
	assertSchemaDigest(t, contract.ConfigSchemaRef, contract.ConfigSchemaDigest)
	for _, operation := range contract.Operations {
		assertSchemaDigest(t, operation.InputSchemaRef, operation.InputSchemaDigest)
		assertSchemaDigest(t, operation.OutputSchemaRef, operation.OutputSchemaDigest)
	}
}

func assertSchemaDigest(t *testing.T, ref, digest string) {
	t.Helper()
	pathByRef := map[string]string{
		"schema://providers/workflow-plugin-stream/video-stream/config/v1":           "../schemas/video-stream-config.schema.json",
		"schema://providers/workflow-plugin-stream/video-stream/operation-input/v1":  "../schemas/video-stream-operation-input.schema.json",
		"schema://providers/workflow-plugin-stream/video-stream/operation-output/v1": "../schemas/video-stream-operation-output.schema.json",
	}
	path, ok := pathByRef[ref]
	if !ok {
		t.Fatalf("unexpected schema ref %q", ref)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema %s: %v", path, err)
	}
	var js any
	if err := json.Unmarshal(data, &js); err != nil {
		t.Fatalf("schema %s is not valid json: %v", path, err)
	}
	sum := sha256.Sum256(data)
	want := "sha256:" + hex.EncodeToString(sum[:])
	if digest != want {
		t.Fatalf("digest for %s = %q, want %q", ref, digest, want)
	}
}

func readJSON(t *testing.T, path string, out any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}
