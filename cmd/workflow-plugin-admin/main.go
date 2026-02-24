// Command workflow-plugin-admin is a workflow engine external plugin that
// serves the admin dashboard UI and injects admin config routes into the host.
// It runs as a subprocess and communicates with the host workflow engine via
// the go-plugin protocol.
package main

import (
	"github.com/GoCodeAlone/workflow-plugin-admin/internal"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func main() {
	sdk.Serve(internal.NewAdminPlugin())
}
