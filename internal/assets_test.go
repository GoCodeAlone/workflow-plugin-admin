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
		`id="contribution-nav"`,
		`id="identity-panel"`,
		`id="authorization-panel"`,
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

func mustReadEmbeddedAsset(t *testing.T, path string) []byte {
	t.Helper()
	data, err := embeddedUIFS.ReadFile(path)
	if err != nil {
		t.Fatalf("read embedded asset %s: %v", path, err)
	}
	return data
}
