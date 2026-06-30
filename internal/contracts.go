package internal

import (
	"github.com/GoCodeAlone/workflow-plugin-compute-core/protocol"
	"github.com/GoCodeAlone/workflow-plugin-stream/catalog"
)

// ProviderContracts returns the public video-stream provider contract catalog.
func ProviderContracts() []protocol.ProviderContract {
	return catalog.ProviderContracts()
}

// VideoStreamProviderContract describes the MediaMTX-backed video-stream provider.
func VideoStreamProviderContract() protocol.ProviderContract {
	return catalog.VideoStreamProviderContract()
}

// RuntimeAdapterContracts returns this plugin's service-session adapter metadata.
func RuntimeAdapterContracts() []protocol.RuntimeAdapterContract {
	return catalog.RuntimeAdapterContracts()
}
