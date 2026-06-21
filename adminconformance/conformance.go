// Package adminconformance provides reusable checks for hosts that mount the
// workflow-plugin-admin shell and contributed runtime-integrated admin surfaces.
package adminconformance

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
)

const defaultContributionsPath = "/api/admin/contributions"
const maxProbeResponseBodyBytes = 1 << 20

// Options configures an admin conformance check. Supply either Handler for
// in-process checks or BaseURL for launched-host checks.
type Options struct {
	Handler http.Handler
	BaseURL string
	Client  *http.Client

	ContributionsPath      string
	AuthenticatedHeaders   map[string]string
	UnauthenticatedHeaders map[string]string

	ExpectedContributions []ExpectedContribution
	RouteProbes           []RouteProbe
	UnauthenticatedProbes []RouteProbe
}

// ExpectedContribution describes an admin surface the host expects the admin
// registry endpoint to expose.
type ExpectedContribution struct {
	ID                string `json:"id"`
	Title             string `json:"title,omitempty"`
	Path              string `json:"path,omitempty"`
	RenderMode        string `json:"render_mode,omitempty"`
	RuntimeIntegrated bool
	BackingRoutes     []RouteProbe
}

// RouteProbe describes one HTTP request/response invariant.
type RouteProbe struct {
	Name           string
	Method         string
	Path           string
	ExpectedStatus int
	RequireBody    bool
	Headers        map[string]string
}

// Result is the aggregate conformance outcome.
type Result struct {
	Pass     bool
	Failures []string
}

type contributionListResponse struct {
	Contributions []ExpectedContribution `json:"contributions"`
}

// Check runs the configured conformance probes.
func Check(options Options) Result {
	options = normalizeOptions(options)
	result := Result{Pass: true}
	if options.Handler == nil && strings.TrimSpace(options.BaseURL) == "" {
		result.fail("adminconformance: Handler or BaseURL is required")
		return result
	}

	checkContributions(&result, options)
	for _, probe := range options.RouteProbes {
		checkRoute(&result, options, probe, options.AuthenticatedHeaders, "")
	}
	for _, probe := range options.UnauthenticatedProbes {
		checkRoute(&result, options, probe, options.UnauthenticatedHeaders, "")
	}
	for _, expected := range options.ExpectedContributions {
		if expected.RuntimeIntegrated && len(expected.BackingRoutes) == 0 {
			result.fail("runtime-integrated contribution %s declares no backing route probes", expected.ID)
			continue
		}
		for _, probe := range expected.BackingRoutes {
			checkRoute(&result, options, probe, options.AuthenticatedHeaders, expected.ID)
		}
	}
	return result
}

func normalizeOptions(options Options) Options {
	if strings.TrimSpace(options.ContributionsPath) == "" {
		options.ContributionsPath = defaultContributionsPath
	}
	if options.Client == nil {
		options.Client = http.DefaultClient
	}
	return options
}

func checkContributions(result *Result, options Options) []ExpectedContribution {
	response := doRequest(result, options, RouteProbe{
		Name:           "admin contributions",
		Method:         http.MethodGet,
		Path:           options.ContributionsPath,
		ExpectedStatus: http.StatusOK,
		RequireBody:    true,
	}, options.AuthenticatedHeaders, "")
	if response == nil {
		return nil
	}
	defer response.Body.Close()
	var payload contributionListResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		result.fail("admin contributions: decode JSON: %v", err)
		return nil
	}
	for _, expected := range options.ExpectedContributions {
		actual, ok := findContribution(payload.Contributions, expected.ID)
		if !ok {
			result.fail("expected contribution %s not listed", expected.ID)
			continue
		}
		if expected.Path != "" && actual.Path != expected.Path {
			result.fail("contribution %s path = %q, want %q", expected.ID, actual.Path, expected.Path)
		}
		if expected.Title != "" && actual.Title != expected.Title {
			result.fail("contribution %s title = %q, want %q", expected.ID, actual.Title, expected.Title)
		}
		if expected.RenderMode != "" && actual.RenderMode != expected.RenderMode {
			result.fail("contribution %s render_mode = %q, want %q", expected.ID, actual.RenderMode, expected.RenderMode)
		}
	}
	return payload.Contributions
}

func findContribution(contributions []ExpectedContribution, id string) (ExpectedContribution, bool) {
	for _, contribution := range contributions {
		if contribution.ID == id {
			return contribution, true
		}
	}
	return ExpectedContribution{}, false
}

func checkRoute(result *Result, options Options, probe RouteProbe, headers map[string]string, contributionID string) {
	response := doRequest(result, options, probe, headers, contributionID)
	if response != nil {
		_ = response.Body.Close()
	}
}

func doRequest(result *Result, options Options, probe RouteProbe, headers map[string]string, contributionID string) *http.Response {
	probe = normalizeProbe(probe)
	req := httptest.NewRequest(probe.Method, probe.Path, nil)
	if options.BaseURL != "" {
		var err error
		req, err = http.NewRequest(probe.Method, strings.TrimRight(options.BaseURL, "/")+probe.Path, nil)
		if err != nil {
			result.fail("%s: build request: %v", probeLabel(probe, contributionID), err)
			return nil
		}
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	for key, value := range probe.Headers {
		req.Header.Set(key, value)
	}

	var response *http.Response
	if options.Handler != nil {
		rec := httptest.NewRecorder()
		options.Handler.ServeHTTP(rec, req)
		response = rec.Result()
	} else {
		var err error
		response, err = options.Client.Do(req)
		if err != nil {
			result.fail("%s: request failed: %v", probeLabel(probe, contributionID), err)
			return nil
		}
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxProbeResponseBodyBytes+1))
	if err != nil {
		result.fail("%s: read body: %v", probeLabel(probe, contributionID), err)
		return response
	}
	response.Body.Close()
	if len(body) > maxProbeResponseBodyBytes {
		result.fail("%s: response body exceeds %d bytes", probeLabel(probe, contributionID), maxProbeResponseBodyBytes)
		body = body[:maxProbeResponseBodyBytes]
	}
	response.Body = io.NopCloser(strings.NewReader(string(body)))

	if response.StatusCode != probe.ExpectedStatus {
		result.fail("%s: status = %d, want %d body=%s", probeLabel(probe, contributionID), response.StatusCode, probe.ExpectedStatus, strings.TrimSpace(string(body)))
	}
	if probe.RequireBody && strings.TrimSpace(string(body)) == "" {
		result.fail("%s: empty body", probeLabel(probe, contributionID))
	}
	return response
}

func normalizeProbe(probe RouteProbe) RouteProbe {
	if strings.TrimSpace(probe.Method) == "" {
		probe.Method = http.MethodGet
	}
	if probe.ExpectedStatus == 0 {
		probe.ExpectedStatus = http.StatusOK
	}
	if !strings.HasPrefix(probe.Path, "/") {
		probe.Path = "/" + strings.TrimLeft(probe.Path, "/")
	}
	if strings.TrimSpace(probe.Name) == "" {
		probe.Name = probe.Method + " " + probe.Path
	}
	return probe
}

func probeLabel(probe RouteProbe, contributionID string) string {
	label := probe.Name
	if label == "" {
		label = probe.Method + " " + probe.Path
	} else {
		label += " " + probe.Method + " " + probe.Path
	}
	if contributionID != "" {
		return fmt.Sprintf("contribution %s %s", contributionID, label)
	}
	return label
}

func (r *Result) fail(format string, args ...any) {
	r.Pass = false
	r.Failures = append(r.Failures, fmt.Sprintf(format, args...))
}
