// Package workflowpluginadmin provides the admin workflow plugin.
package workflowpluginadmin

import (
	"github.com/GoCodeAlone/workflow-plugin-admin/internal"
	"github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

// NewAdminPlugin returns the admin SDK plugin provider.
func NewAdminPlugin() sdk.PluginProvider {
	return internal.NewAdminPlugin()
}
