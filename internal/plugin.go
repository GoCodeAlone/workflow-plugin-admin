package internal

import (
	"fmt"
	"os"
	"path/filepath"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
	"gopkg.in/yaml.v3"
)

// adminPlugin implements PluginProvider and ConfigProvider.
type adminPlugin struct{}

// NewAdminPlugin returns a new adminPlugin instance.
func NewAdminPlugin() sdk.PluginProvider {
	return &adminPlugin{}
}

// Manifest returns plugin metadata.
func (p *adminPlugin) Manifest() sdk.PluginManifest {
	return sdk.PluginManifest{
		Name:        "admin",
		Version:     "1.0.0",
		Author:      "GoCodeAlone",
		Description: "Admin dashboard UI and config-driven admin routes",
	}
}

// ConfigFragment returns the embedded admin config.yaml with absolute paths
// resolved relative to the plugin's working directory.
func (p *adminPlugin) ConfigFragment() ([]byte, error) {
	if err := extractAssets(); err != nil {
		return nil, fmt.Errorf("admin plugin: extract assets: %w", err)
	}

	dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("admin plugin: get working directory: %w", err)
	}

	absUIPath := filepath.Join(dir, "ui_dist")

	var cfg map[string]any
	if err := yaml.Unmarshal(configData, &cfg); err != nil {
		return nil, fmt.Errorf("admin plugin: parse config: %w", err)
	}

	// Update static.fileserver and admin.dashboard root paths to absolute paths.
	if modules, ok := cfg["modules"].([]any); ok {
		for _, m := range modules {
			mod, ok := m.(map[string]any)
			if !ok {
				continue
			}
			modType, _ := mod["type"].(string)
			if modType == "static.fileserver" || modType == "admin.dashboard" {
				if config, ok := mod["config"].(map[string]any); ok {
					config["root"] = absUIPath
				}
			}
		}
	}

	return yaml.Marshal(cfg)
}
