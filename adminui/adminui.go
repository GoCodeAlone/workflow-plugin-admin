// Package adminui exposes the admin plugin's embedded shell for Go hosts.
package adminui

import (
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/GoCodeAlone/workflow-plugin-admin/internal"
)

const (
	defaultContributionsEndpoint = "/api/admin/contributions"
	defaultLoginEndpoint         = "/api/admin/auth/login"
	defaultTokenStorageKey       = "workflow.admin.token"
	defaultShellMarker           = `<div class="shell" data-admin-shell-version="1" data-contributions-endpoint="/api/admin/contributions" data-login-endpoint="/api/admin/auth/login" data-token-storage-key="workflow.admin.token">`
)

// AuthMode controls how the browser shell obtains admin credentials.
type AuthMode string

const (
	// AuthModeBearer stores a bearer token in localStorage and sends it to admin APIs.
	AuthModeBearer AuthMode = "bearer"
	// AuthModeSession lets the host provide credentials through same-origin cookies.
	AuthModeSession AuthMode = "session"
)

// ShellOptions configures the embedded admin shell for a host application.
type ShellOptions struct {
	ContributionsEndpoint string
	LoginEndpoint         string
	TokenStorageKey       string
	AuthMode              AuthMode
}

// Handler serves the embedded admin shell with the supplied host options.
func Handler(options ShellOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		html, err := ShellHTML(options)
		if err != nil {
			http.Error(w, "admin shell unavailable", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.Method == http.MethodHead {
			return
		}
		_, _ = w.Write(html)
	})
}

// ShellHTML renders the embedded shell index with host-specific endpoints.
func ShellHTML(options ShellOptions) ([]byte, error) {
	options = normalizeOptions(options)
	data, err := internal.ReadEmbeddedUIAsset("ui_dist/index.html")
	if err != nil {
		return nil, err
	}
	replacement := shellMarker(options)
	rendered := strings.Replace(string(data), defaultShellMarker, replacement, 1)
	if rendered == string(data) {
		return nil, fmt.Errorf("admin shell marker not found")
	}
	return []byte(rendered), nil
}

func normalizeOptions(options ShellOptions) ShellOptions {
	if options.ContributionsEndpoint == "" {
		options.ContributionsEndpoint = defaultContributionsEndpoint
	}
	if options.LoginEndpoint == "" {
		options.LoginEndpoint = defaultLoginEndpoint
	}
	if options.AuthMode == "" {
		options.AuthMode = AuthModeBearer
	}
	if options.AuthMode != AuthModeSession && options.TokenStorageKey == "" {
		options.TokenStorageKey = defaultTokenStorageKey
	}
	return options
}

func shellMarker(options ShellOptions) string {
	attrs := []string{
		`class="shell"`,
		`data-admin-shell-version="1"`,
		`data-auth-mode="` + attr(string(options.AuthMode)) + `"`,
		`data-contributions-endpoint="` + attr(options.ContributionsEndpoint) + `"`,
		`data-login-endpoint="` + attr(options.LoginEndpoint) + `"`,
	}
	if options.TokenStorageKey != "" {
		attrs = append(attrs, `data-token-storage-key="`+attr(options.TokenStorageKey)+`"`)
	}
	return "<div " + strings.Join(attrs, " ") + ">"
}

func attr(value string) string {
	return html.EscapeString(value)
}
