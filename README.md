# workflow-plugin-stream

Live video ingest, multiplex, and restream provider for Workflow, backed by
MediaMTX runtime manifests and the compute-core video-stream contract.

This plugin is the video-stream provider surface for
`workflow-compute` issue #241. It publishes the provider and runtime metadata
needed by compute runtimes, plus two direct Workflow step types for stream
session orchestration.

## Installation

```sh
wfctl plugin install workflow-plugin-stream
```

The public registry entry is `workflow-plugin-stream`. The latest released
registry version before this docs release is `v0.1.2`; this README is intended
to ship with the `v0.1.3` patch release and the matching registry sync.

## Contract Files

- `plugin.json` exposes the external plugin metadata and direct step types:
  `stream.start` and `stream.restream`.
- `stream-providers.json` publishes provider contract
  `workflow-plugin-stream.video-stream.v1` for provider id `video-stream`.
- `runtime-adapters.json` publishes runtime adapter `stream-service-session`.
- `managed-runtime-bundles.json` pins the MediaMTX runtime bundle metadata.
- `schemas/video-stream-*.schema.json` describe provider config and operation
  input/output shapes.

The provider contract is additive metadata for compute runtimes. It advertises:

- workload kind `video-stream`;
- operating mode `warm-service`;
- executor provider `stream-service-session`;
- execution security tier `hardened-container`;
- proof tier `stream-segment-manifest`;
- network modes `direct` and `relay`; and
- operations `start_stream`, `set_transform`, `add_destination`,
  `remove_destination`, `add_rendition`, and `stop_stream`.

## Runtime Adapter

`session/adapter.go` implements the compute-core service-session adapter. It
accepts `video-stream` workloads, starts a supervised service session, supports
lease renewal, and mutates live destination state without restarting the
session. The runtime descriptor matches `runtime-adapters.json`:

- adapter id `stream-service-session`;
- runtime profile `service-oci-v1`;
- conformance profile `mediamtx-service-session-v1`;
- execution security tier `hardened-container`; and
- proof tier `stream-segment-manifest`.

Host-owned responsibilities stay outside this plugin: task admission, leases,
authorization, credential resolution, proof/reward mutation, and worker binding.

## Direct Steps And Compute Dispatch

The plugin registers two direct Workflow step types:

- `stream.start` starts a live ingest/multiplex session.
- `stream.restream` adds or updates restream destinations for an existing live
  session.

These direct steps are useful when an application wants to address the stream
plugin explicitly. Compute workloads should normally use `step.compute_stream`
from `workflow-plugin-compute`; that dispatch step selects the registered
`video-stream` provider and uses the provider/runtime contracts in this repo.

In the current release line, the direct external-plugin steps are deliberately
thin and return `status: pending-provider-contract` until the host runtime wires
provider dispatch. The provider contract, runtime adapter, ingest helper,
auth-hook logic, and stream proof manifest code are present for host/runtime
integration.

## Ingest And Auth

`stream.BuildIngestDescriptor` creates publisher endpoints for a host-issued
lease scope. Supported ingest protocols are:

- RTMP;
- SRT; and
- WHIP/WebRTC publishing through the MediaMTX auth hook path.

The descriptor returns a `publish_token_ref` such as
`secret://stream/publish/live-show-01`; it does not return raw publish tokens.
`stream.AuthHook` validates MediaMTX publish requests against host-issued token
claims and the active lease scope. Denial reasons redact credentials.

## Transform And Restream Operations

The provider contract advertises live operations for session mutation:

- `set_transform` for changing active transform state;
- `add_destination` and `remove_destination` for restream target changes;
- `add_rendition` for adaptive-output lanes; and
- `stop_stream` for controlled shutdown.

Current runtime mutation support covers live destination add/remove through
`session.Mutation`. Pan/crop/resolution/framerate transforms and rendition
materialization are contract-level lanes for the compute/runtime integration
phases; they are not implemented as a standalone ffmpeg pipeline in this plugin.

## Proofs

The proof tier is `stream-segment-manifest`. `mediamtx.BuildStreamManifest`
builds compute-core `StreamManifest` proofs from captured segment files,
liveness nonces, sampled frames, delivery receipts, and destination byte counts.
The manifest records segment hashes and delivery evidence without exposing
publish credentials.

## Example Config

```yaml
plugins:
  stream:
    source: workflow-plugin-stream

pipelines:
  live_video:
    steps:
      - id: start_stream
        type: step.compute_stream
        with:
          provider_id: video-stream
          operation: start_stream
          stream_spec:
            ingest_protocols: [rtmp, srt, whip]
            max_connections: 2
            codecs: [h264, aac]
          lease:
            path_name: live/show-01
            ingest_host: ingest.example.com
            publish_token_ref: secret://stream/publish/live-show-01

      - id: add_destination
        type: step.compute_stream
        with:
          provider_id: video-stream
          operation: add_destination
          stream_handle: ${steps.start_stream.output.stream_handle}
          destination:
            id: archive
            url_ref: secret://stream/restream/archive-url
```

For direct plugin use, replace `step.compute_stream` with `stream.start` or
`stream.restream` and keep credential values behind `secret://` references.

## Limitations

- The plugin does not implement edge-CDN geo routing; use provider/CDN plugins
  for CDN-specific policy and delivery configuration.
- The plugin does not mint or store publish/restream secrets. Hosts resolve
  `secret://` refs and pass only scoped claims or destination refs.
- The direct external-plugin steps are not a full local MediaMTX process runner;
  runtime execution is expected through the compute provider/session adapter.
- The managed bundle metadata currently targets Linux amd64 MediaMTX
  `v1.19.1`.

## Development

```sh
# Build
make build

# Test
make test

# Install locally
make install-local
```

## Module

Go module: `github.com/GoCodeAlone/workflow-plugin-stream`
