# workflow-plugin-stream

Live video ingest/multiplex/restream (MediaMTX)

## Installation

```sh
wfctl plugin install workflow-plugin-stream
```

## Development

```sh
# Build
make build

# Test
make test

# Install locally
make install-local
```

## Step Types

- `stream.start` — Start a live ingest/multiplex session.
- `stream.restream` — Add or update restream destinations for a live session.

## Module

Go module: `github.com/GoCodeAlone/workflow-plugin-stream`
