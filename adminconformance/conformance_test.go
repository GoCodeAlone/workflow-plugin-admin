package adminconformance

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckConformancePassesForMountedAdminSurface(t *testing.T) {
	handler := testAdminHandler(t, true)

	result := Check(Options{
		Handler:              handler,
		AuthenticatedHeaders: map[string]string{"Authorization": "Bearer admin"},
		ExpectedContributions: []ExpectedContribution{
			{
				ID:                "authz",
				Path:              "/admin/authz",
				RuntimeIntegrated: true,
				BackingRoutes: []RouteProbe{
					{Name: "authz roles api", Method: http.MethodGet, Path: "/api/admin/authz/roles", ExpectedStatus: http.StatusOK, RequireBody: true},
				},
			},
		},
		RouteProbes: []RouteProbe{
			{Name: "admin shell", Method: http.MethodGet, Path: "/admin", ExpectedStatus: http.StatusOK, RequireBody: true},
		},
		UnauthenticatedProbes: []RouteProbe{
			{Name: "contributions auth gate", Method: http.MethodGet, Path: "/api/admin/contributions", ExpectedStatus: http.StatusUnauthorized},
		},
	})

	if !result.Pass {
		t.Fatalf("conformance failed: %#v", result.Failures)
	}
}

func TestCheckConformanceNamesMissingContributionRoute(t *testing.T) {
	handler := testAdminHandler(t, false)

	result := Check(Options{
		Handler:              handler,
		AuthenticatedHeaders: map[string]string{"Authorization": "Bearer admin"},
		ExpectedContributions: []ExpectedContribution{
			{
				ID:                "authz",
				Path:              "/admin/authz",
				RuntimeIntegrated: true,
				BackingRoutes: []RouteProbe{
					{Name: "authz roles api", Method: http.MethodGet, Path: "/api/admin/authz/roles", ExpectedStatus: http.StatusOK, RequireBody: true},
				},
			},
		},
	})

	if result.Pass {
		t.Fatal("conformance passed with missing backing route")
	}
	joined := strings.Join(result.Failures, "\n")
	if !strings.Contains(joined, "authz") || !strings.Contains(joined, "/api/admin/authz/roles") {
		t.Fatalf("failure does not name contribution and route: %#v", result.Failures)
	}
}

func TestCheckConformanceFailsWhenRuntimeContributionDeclaresNoBackingRoutes(t *testing.T) {
	handler := testAdminHandler(t, true)

	result := Check(Options{
		Handler:              handler,
		AuthenticatedHeaders: map[string]string{"Authorization": "Bearer admin"},
		ExpectedContributions: []ExpectedContribution{
			{ID: "authz", Path: "/admin/authz", RuntimeIntegrated: true},
		},
	})

	if result.Pass {
		t.Fatal("conformance passed for runtime-integrated contribution without backing routes")
	}
	if !strings.Contains(strings.Join(result.Failures, "\n"), "runtime-integrated contribution authz declares no backing route probes") {
		t.Fatalf("failure did not name missing backing probes: %#v", result.Failures)
	}
}

func testAdminHandler(t *testing.T, includeBackingRoute bool) http.Handler {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<div data-admin-shell-version="1">Admin</div>`))
	})
	mux.HandleFunc("/api/admin/contributions", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"contributions": []map[string]any{
				{"id": "authz", "title": "Authorization", "path": "/admin/authz", "render_mode": "iframe"},
			},
		})
	})
	if includeBackingRoute {
		mux.HandleFunc("/api/admin/authz/roles", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"name":"admin"}]`))
		})
	}
	return mux
}

func TestCheckConformanceWorksWithHTTPServer(t *testing.T) {
	server := httptest.NewServer(testAdminHandler(t, true))
	defer server.Close()

	result := Check(Options{
		BaseURL:              server.URL,
		Client:               server.Client(),
		AuthenticatedHeaders: map[string]string{"Authorization": "Bearer admin"},
		ExpectedContributions: []ExpectedContribution{
			{
				ID:                "authz",
				Path:              "/admin/authz",
				RuntimeIntegrated: true,
				BackingRoutes: []RouteProbe{
					{Name: "authz roles api", Method: http.MethodGet, Path: "/api/admin/authz/roles", ExpectedStatus: http.StatusOK, RequireBody: true},
				},
			},
		},
	})

	if !result.Pass {
		t.Fatalf("conformance failed: %#v", result.Failures)
	}
}
