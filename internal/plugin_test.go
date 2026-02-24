package internal

import (
	"testing"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func TestNewAdminPlugin(t *testing.T) {
	plugin := NewAdminPlugin()
	if plugin == nil {
		t.Fatal("NewAdminPlugin() returned nil")
	}

	var _ sdk.PluginProvider = plugin
}

func TestAdminPlugin_Manifest(t *testing.T) {
	plugin := NewAdminPlugin()
	manifest := plugin.Manifest()

	if manifest.Name != "admin" {
		t.Errorf("expected name 'admin', got %q", manifest.Name)
	}
	if manifest.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", manifest.Version)
	}
	if manifest.Author != "GoCodeAlone" {
		t.Errorf("expected author 'GoCodeAlone', got %q", manifest.Author)
	}
	if manifest.Description == "" {
		t.Error("expected non-empty description")
	}
}

func TestAdminPlugin_ConfigFragment(t *testing.T) {
	plugin := NewAdminPlugin()

	cp, ok := plugin.(sdk.ConfigProvider)
	if !ok {
		t.Fatal("plugin does not implement ConfigProvider")
	}

	data, err := cp.ConfigFragment()
	if err != nil {
		t.Fatalf("ConfigFragment() error = %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty config fragment")
	}
}

func TestAdminPlugin_Interfaces(t *testing.T) {
	plugin := NewAdminPlugin()

	var _ sdk.PluginProvider = plugin

	if _, ok := plugin.(sdk.ConfigProvider); !ok {
		t.Error("plugin does not implement ConfigProvider")
	}

	// Must NOT implement ModuleProvider (all admin modules are native host types)
	if _, ok := plugin.(sdk.ModuleProvider); ok {
		t.Error("plugin should not implement ModuleProvider")
	}
}
