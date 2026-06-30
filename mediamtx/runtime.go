package mediamtx

import (
	"errors"
	"fmt"

	core "github.com/GoCodeAlone/workflow-plugin-compute-core/protocol"
	"gopkg.in/yaml.v3"
)

// ConfigOptions carries host-provided values that do not belong in StreamSpec.
type ConfigOptions struct {
	PathName        string
	AuthHTTPAddress string
}

// RenderConfig produces a minimal MediaMTX configuration for a stream session.
func RenderConfig(spec core.StreamSpec, opts ConfigOptions) ([]byte, error) {
	if opts.PathName == "" {
		return nil, errors.New("path name is required")
	}
	if opts.AuthHTTPAddress == "" {
		return nil, errors.New("auth HTTP address is required")
	}
	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("stream spec: %w", err)
	}

	var ingest ingestProtocols
	for _, protocol := range spec.IngestProtocols {
		switch protocol {
		case "rtmp":
			ingest.rtmp = true
		case "srt":
			ingest.srt = true
		case "whip":
			ingest.whip = true
		default:
			return nil, fmt.Errorf("unsupported ingest protocol %q", protocol)
		}
	}

	cfg := mediaMTXConfig{
		LogLevel:        "info",
		LogDestinations: []string{"stdout"},
		AuthMethod:      "http",
		AuthHTTPAddress: opts.AuthHTTPAddress,
		AuthHTTPExclude: []authHTTPExclude{
			{Action: "api"},
			{Action: "metrics"},
			{Action: "pprof"},
		},
		API:           true,
		APIAddress:    ":9997",
		RTSP:          false,
		RTMP:          ingest.rtmp,
		RTMPAddress:   ":1935",
		SRT:           ingest.srt,
		SRTAddress:    ":8890",
		WebRTC:        ingest.whip || spec.ViewerEgress.WHEP,
		WebRTCAddress: ":8889",
		HLS:           spec.ViewerEgress.HLS,
		HLSAddress:    ":8888",
		HLSVariant:    "lowLatency",
		Paths: map[string]pathConfig{
			opts.PathName: {
				Source:            "publisher",
				Record:            spec.Recording,
				OverridePublisher: true,
			},
		},
	}

	return yaml.Marshal(cfg)
}

type ingestProtocols struct {
	rtmp bool
	srt  bool
	whip bool
}

type mediaMTXConfig struct {
	LogLevel        string                `yaml:"logLevel"`
	LogDestinations []string              `yaml:"logDestinations"`
	AuthMethod      string                `yaml:"authMethod"`
	AuthHTTPAddress string                `yaml:"authHTTPAddress"`
	AuthHTTPExclude []authHTTPExclude     `yaml:"authHTTPExclude"`
	API             bool                  `yaml:"api"`
	APIAddress      string                `yaml:"apiAddress"`
	RTSP            bool                  `yaml:"rtsp"`
	RTMP            bool                  `yaml:"rtmp"`
	RTMPAddress     string                `yaml:"rtmpAddress"`
	SRT             bool                  `yaml:"srt"`
	SRTAddress      string                `yaml:"srtAddress"`
	WebRTC          bool                  `yaml:"webrtc"`
	WebRTCAddress   string                `yaml:"webrtcAddress"`
	HLS             bool                  `yaml:"hls"`
	HLSAddress      string                `yaml:"hlsAddress"`
	HLSVariant      string                `yaml:"hlsVariant,omitempty"`
	Paths           map[string]pathConfig `yaml:"paths"`
}

type authHTTPExclude struct {
	Action string `yaml:"action"`
}

type pathConfig struct {
	Source            string `yaml:"source"`
	Record            bool   `yaml:"record"`
	OverridePublisher bool   `yaml:"overridePublisher"`
}
