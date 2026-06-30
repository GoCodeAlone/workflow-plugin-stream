package internal

import core "github.com/GoCodeAlone/workflow-plugin-compute-core/protocol"

const (
	streamContractVersion = "v0.1.0"

	streamConfigSchemaRef          = "schema://providers/workflow-plugin-stream/video-stream/config/v1"
	streamConfigSchemaDigest       = "sha256:6b4b607a5046db8f18935c2cb34029c9db807de2f33a22ba91c1196c418e255a"
	streamOperationInputSchemaRef  = "schema://providers/workflow-plugin-stream/video-stream/operation-input/v1"
	streamOperationInputDigest     = "sha256:80756f2b391b002cfd906bd0b973f1d1e3c08be08632bbaa60e264f1b737dfe5"
	streamOperationOutputSchemaRef = "schema://providers/workflow-plugin-stream/video-stream/operation-output/v1"
	streamOperationOutputDigest    = "sha256:f9856101666889d58339872f3f45b9b0cbb5a5e3d9367bb7936cb4c1fc27d30f"

	streamExecutorProvider   = "stream-service-session"
	streamConformanceProfile = "mediamtx-service-session-v1"
)

// ProviderContracts returns the public video-stream provider contract catalog.
func ProviderContracts() []core.ProviderContract {
	return []core.ProviderContract{VideoStreamProviderContract()}
}

// VideoStreamProviderContract describes the MediaMTX-backed video-stream provider.
func VideoStreamProviderContract() core.ProviderContract {
	tier := core.ExecutionHardenedContainer
	proof := core.ProofStreamSegmentManifest
	runtimeProfile := core.DefaultProviderRuntimeProfile("service-sandboxed-container", tier, proof)
	runtimeProfile.ID = streamExecutorProvider + "-" + string(tier) + "-" + string(proof) + "-runtime"
	runtimeProfile.ExecutorProvider = streamExecutorProvider
	runtimeProfile.ConformanceProfiles = append(runtimeProfile.ConformanceProfiles, streamConformanceProfile)

	return core.ProviderContract{
		ProtocolVersion:    core.Version,
		ID:                 "workflow-plugin-stream.video-stream.v1",
		PluginID:           "workflow-plugin-stream",
		ProviderID:         "video-stream",
		ContractID:         "workflow-plugin-stream.video-stream.v1",
		Version:            streamContractVersion,
		DisplayName:        "MediaMTX video-stream provider",
		ConfigSchemaRef:    streamConfigSchemaRef,
		ConfigSchemaDigest: streamConfigSchemaDigest,
		OperatingModes: []core.NetworkOperatingMode{
			core.NetworkModeWarmService,
		},
		WorkloadKinds: []string{string(core.WorkloadVideoStream)},
		ExecutorProviders: []string{
			streamExecutorProvider,
		},
		ExecutionSecurityTiers: []core.ExecutionSecurityTier{
			tier,
		},
		ProofTiers: []core.ProofTier{
			proof,
		},
		NetworkModes: []core.NetworkMode{
			core.NetworkModeDirect,
			core.NetworkModeRelay,
		},
		Operations: providerOperations(
			"start_stream",
			"set_transform",
			"add_destination",
			"remove_destination",
			"add_rendition",
			"stop_stream",
		),
		RuntimeContract: core.ProviderRuntimeContract{
			Profiles: []core.ProviderRuntimeProfile{runtimeProfile},
		},
	}
}

// RuntimeAdapterContracts returns this plugin's service-session adapter metadata.
func RuntimeAdapterContracts() []core.RuntimeAdapterContract {
	return []core.RuntimeAdapterContract{{
		ProtocolVersion: core.Version,
		AdapterID:       streamExecutorProvider,
		Descriptor: core.RuntimeDescriptor{
			Name:                  streamExecutorProvider,
			Version:               streamContractVersion,
			ExecutionSecurityTier: core.ExecutionHardenedContainer,
			ProofTier:             core.ProofStreamSegmentManifest,
			ImageDigest:           "sha256:035ee04f91b1c7a0c02e13b2139ca2456e43b6bd6a80e3100e8c228556e07807",
			RootFSDigest:          "sha256:a008c9a89b4040a3f6903df99616dcf46c68ef619ee6ef204e10c60455eccf6f",
		},
		Kinds: []core.RuntimeAdapterKind{
			core.RuntimeAdapterServiceSession,
		},
		WorkloadKinds: []core.WorkloadKind{
			core.WorkloadVideoStream,
		},
		RuntimeProfiles: []core.RuntimeProfile{
			core.RuntimeProfileServiceOCI,
		},
		WorkspacePolicy: core.RuntimeWorkspaceUnavailable,
		ConformanceProfiles: []string{
			streamConformanceProfile,
		},
	}}
}

func providerOperations(ids ...string) []core.ProviderOperation {
	operations := make([]core.ProviderOperation, 0, len(ids))
	for _, id := range ids {
		operations = append(operations, core.ProviderOperation{
			ID:                 id,
			InputSchemaRef:     streamOperationInputSchemaRef,
			InputSchemaDigest:  streamOperationInputDigest,
			OutputSchemaRef:    streamOperationOutputSchemaRef,
			OutputSchemaDigest: streamOperationOutputDigest,
		})
	}
	return operations
}
