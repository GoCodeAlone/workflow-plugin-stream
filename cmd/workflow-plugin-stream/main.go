package main

import (
	"fmt"
	"os"

	"github.com/GoCodeAlone/workflow-plugin-stream/internal"
	"github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		fmt.Println("workflow-plugin-stream: Workflow external plugin for live video ingest, multiplex, and restream.")
		return
	}
	sdk.Serve(internal.NewStreamProvider(),
		sdk.WithBuildVersion(sdk.ResolveBuildVersion(internal.Version)),
	)
}
