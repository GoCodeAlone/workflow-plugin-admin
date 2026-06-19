package adminui_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoCodeAlone/workflow-plugin-admin/adminui"
)

func TestShellHTMLDefaultsToBearerAuth(t *testing.T) {
	html, err := adminui.ShellHTML(adminui.ShellOptions{})
	if err != nil {
		t.Fatalf("ShellHTML: %v", err)
	}
	text := string(html)
	for _, needle := range []string{
		`data-admin-shell-version="1"`,
		`data-auth-mode="bearer"`,
		`data-contributions-endpoint="/api/admin/contributions"`,
		`data-login-endpoint="/api/admin/auth/login"`,
		`data-token-storage-key="workflow.admin.token"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("default shell missing %q", needle)
		}
	}
}

func TestShellHTMLCanUseHostSessionAuth(t *testing.T) {
	html, err := adminui.ShellHTML(adminui.ShellOptions{
		AuthMode:              adminui.AuthModeSession,
		ContributionsEndpoint: "/api/admin/contributions",
		LoginEndpoint:         "/login",
		TokenStorageKey:       "",
	})
	if err != nil {
		t.Fatalf("ShellHTML: %v", err)
	}
	text := string(html)
	for _, needle := range []string{
		`data-auth-mode="session"`,
		`data-contributions-endpoint="/api/admin/contributions"`,
		`data-login-endpoint="/login"`,
		`const requiresBearerToken = shell.dataset.authMode !== 'session';`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("session shell missing %q", needle)
		}
	}
	if strings.Contains(text, `data-token-storage-key=""`) {
		t.Fatal("session shell must omit an empty token storage key")
	}
}

func TestHandlerServesConfiguredShell(t *testing.T) {
	handler := adminui.Handler(adminui.ShellOptions{
		AuthMode:              adminui.AuthModeSession,
		ContributionsEndpoint: "/custom/contributions",
		LoginEndpoint:         "/login",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "https://admin.example.test/", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", got)
	}
	for _, needle := range []string{
		`data-auth-mode="session"`,
		`data-contributions-endpoint="/custom/contributions"`,
	} {
		if !strings.Contains(rec.Body.String(), needle) {
			t.Fatalf("handler shell missing %q", needle)
		}
	}
}
