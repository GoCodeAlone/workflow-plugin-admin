package internal

import (
	"strings"
	"testing"
)

func TestEmbeddedAdminShell(t *testing.T) {
	html := string(mustReadEmbeddedAsset(t, "ui_dist/index.html"))
	required := []string{
		`data-admin-shell-version`,
		`data-contributions-endpoint`,
		`data-login-endpoint`,
		`data-token-storage-key`,
		`id="login-form"`,
		`id="contribution-nav"`,
		`id="contribution-list"`,
		`Authorization`,
		`workflow.admin.auth.request`,
		`workflow.admin.auth.response`,
		`event.origin !== window.location.origin`,
		`startsWith('/admin')`,
		`render_mode === 'config-form'`,
		`renderConfigForm`,
		`validate_path`,
		`fetchConfigDescription`,
	}
	for _, needle := range required {
		if !strings.Contains(html, needle) {
			t.Fatalf("embedded admin shell missing %q", needle)
		}
	}
	forbidden := []string{
		"DIGITALOCEAN_TOKEN",
		"JWT_SECRET",
		"PERMIT_API_KEY",
	}
	for _, needle := range forbidden {
		if strings.Contains(html, needle) {
			t.Fatalf("embedded admin shell must not hardcode secret name %q", needle)
		}
	}
}

func TestEmbeddedAdminShellUsesUserFacingToolLanguage(t *testing.T) {
	html := string(mustReadEmbeddedAsset(t, "ui_dist/index.html"))
	required := []string{
		"Admin tools",
		"Management pages",
	}
	for _, needle := range required {
		if !strings.Contains(html, needle) {
			t.Fatalf("embedded admin shell missing user-facing copy %q", needle)
		}
	}
	forbidden := []string{
		"Surfaces",
		"Registered surfaces",
		"Active surface",
		"Contributions",
		"contributed admin surfaces",
	}
	for _, needle := range forbidden {
		if strings.Contains(html, needle) {
			t.Fatalf("embedded admin shell exposes implementation jargon %q", needle)
		}
	}
}

func TestEmbeddedAdminShellDoesNotHardcodePluginSurfaces(t *testing.T) {
	html := string(mustReadEmbeddedAsset(t, "ui_dist/index.html"))
	forbidden := []string{
		`data-panel="identity-panel"`,
		`data-panel="authorization-panel"`,
		`id="identity-panel"`,
		`id="authorization-panel"`,
		`Identity provider`,
		`Authorization mode`,
		`Manage application users through the configured auth plugin.`,
		`Review and mutate roles, permissions, and relations through authz workflows.`,
	}
	for _, needle := range forbidden {
		if strings.Contains(html, needle) {
			t.Fatalf("embedded admin shell hardcodes plugin surface %q", needle)
		}
	}
}

func TestEmbeddedAdminShellRendersContributionDataSafely(t *testing.T) {
	html := string(mustReadEmbeddedAsset(t, "ui_dist/index.html"))
	forbidden := []string{
		"row.innerHTML",
		"table.innerHTML",
		"insertAdjacentHTML",
		"document.write",
	}
	for _, needle := range forbidden {
		if strings.Contains(html, needle) {
			t.Fatalf("embedded admin shell uses unsafe dynamic rendering primitive %q", needle)
		}
	}
	required := []string{
		"document.createTextNode",
		"appendTableCell",
	}
	for _, needle := range required {
		if !strings.Contains(html, needle) {
			t.Fatalf("embedded admin shell missing safe rendering helper %q", needle)
		}
	}
}

func mustReadEmbeddedAsset(t *testing.T, path string) []byte {
	t.Helper()
	data, err := embeddedUIFS.ReadFile(path)
	if err != nil {
		t.Fatalf("read embedded asset %s: %v", path, err)
	}
	return data
}
