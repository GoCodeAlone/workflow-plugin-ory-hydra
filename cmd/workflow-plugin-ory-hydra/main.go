package main

import (
	"github.com/GoCodeAlone/workflow-plugin-ory-hydra/internal"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func main() {
	sdk.Serve(internal.NewOryHydraPlugin(), sdk.WithBuildVersion(sdk.ResolveBuildVersion(internal.Version)))
}
