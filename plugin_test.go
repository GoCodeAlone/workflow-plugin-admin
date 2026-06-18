package workflowpluginadmin_test

import (
	"testing"

	workflowpluginadmin "github.com/GoCodeAlone/workflow-plugin-admin"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func TestNewAdminPluginIsPubliclyImportable(t *testing.T) {
	plugin := workflowpluginadmin.NewAdminPlugin()
	if plugin == nil {
		t.Fatal("NewAdminPlugin() returned nil")
	}
	if _, ok := plugin.(sdk.ModuleProvider); !ok {
		t.Fatal("NewAdminPlugin() must expose the admin dashboard module provider")
	}
	if _, ok := plugin.(sdk.StepProvider); !ok {
		t.Fatal("NewAdminPlugin() must expose admin step providers")
	}
}
