package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/GoCodeAlone/workflow-plugin-stream/internal"
	"github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "conformance" {
		if err := runConformance(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		return
	}
	if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		fmt.Println("workflow-plugin-stream: Workflow external plugin for live video ingest, multiplex, and restream.")
		return
	}
	sdk.Serve(internal.NewStreamProvider(),
		sdk.WithBuildVersion(sdk.ResolveBuildVersion(internal.Version)),
	)
}

func runConformance(args []string) error {
	fs := flag.NewFlagSet("conformance", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	ingest := fs.String("ingest", "", "ingest protocol")
	egress := fs.String("egress", "", "egress protocol")
	artifact := fs.String("artifact", "", "conformance artifact path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !supported(*ingest, "rtmp", "srt", "whip") {
		return fmt.Errorf("unsupported ingest protocol %q", *ingest)
	}
	if !supported(*egress, "hls", "whep") {
		return fmt.Errorf("unsupported egress protocol %q", *egress)
	}
	if strings.TrimSpace(*artifact) == "" {
		return errors.New("artifact is required")
	}
	data, err := os.ReadFile(*artifact)
	if err != nil {
		return fmt.Errorf("read artifact: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return errors.New("artifact is empty")
	}
	var js any
	if err := json.Unmarshal(data, &js); err != nil {
		return fmt.Errorf("artifact must be JSON: %w", err)
	}
	fmt.Printf("conformance ok ingest=%s egress=%s artifact=%s\n", *ingest, *egress, *artifact)
	return nil
}

func supported(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
