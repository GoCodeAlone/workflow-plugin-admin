package internal

import (
	"context"
	"fmt"

	"github.com/GoCodeAlone/workflow-plugin-admin/internal/contracts"
)

type dashboardModule struct {
	name     string
	config   *contracts.AdminDashboardConfig
	registry *contributionRegistry
}

func newDashboardModule(name string, config *contracts.AdminDashboardConfig) *dashboardModule {
	if config == nil {
		config = &contracts.AdminDashboardConfig{}
	}
	if config.RoutePrefix == "" {
		config.RoutePrefix = "/admin"
	}
	return &dashboardModule{
		name:     name,
		config:   config,
		registry: newContributionRegistry(),
	}
}

func (m *dashboardModule) Init() error {
	return nil
}

func (m *dashboardModule) Start(context.Context) error {
	return nil
}

func (m *dashboardModule) Stop(context.Context) error {
	return nil
}

func (m *dashboardModule) InvokeMethod(method string, args map[string]any) (map[string]any, error) {
	switch method {
	case "RegisterContribution":
		contribution := contributionFromMap(args)
		if err := m.registry.register(contribution); err != nil {
			return nil, err
		}
		return map[string]any{
			"registered":   true,
			"contribution": contributionToMap(contribution),
		}, nil
	case "ListContributions":
		appContext, _ := args["app_context"].(string)
		contributions := m.registry.list(appContext)
		out := make([]map[string]any, 0, len(contributions))
		for _, contribution := range contributions {
			out = append(out, contributionToMap(contribution))
		}
		return map[string]any{"contributions": out}, nil
	case "AuthorizeAction":
		allowed, reason := authorizeFromEvidence(args)
		return map[string]any{
			"allowed":  allowed,
			"subject":  stringValue(args, "subject"),
			"resource": stringValue(args, "resource"),
			"action":   stringValue(args, "action"),
			"reason":   reason,
		}, nil
	case "DispatchResourceAction":
		allowed, reason := authorizeFromEvidence(args)
		if !allowed {
			return map[string]any{
				"result": map[string]any{
					"accepted": false,
					"status":   "denied",
					"error":    reason,
				},
			}, nil
		}
		return map[string]any{
			"result": map[string]any{
				"accepted": true,
				"status":   "accepted",
			},
		}, nil
	default:
		return nil, fmt.Errorf("admin dashboard method %q is not supported", method)
	}
}

func contributionFromMap(args map[string]any) *contracts.AdminContribution {
	if nested, ok := args["contribution"].(map[string]any); ok {
		args = nested
	}
	return &contracts.AdminContribution{
		Id:         stringValue(args, "id"),
		Title:      stringValue(args, "title"),
		Category:   stringValue(args, "category"),
		Path:       stringValue(args, "path"),
		RenderMode: stringValue(args, "render_mode"),
		AppContext: stringValue(args, "app_context"),
		Actions:    stringSliceValue(args, "actions"),
	}
}

func contributionToMap(contribution *contracts.AdminContribution) map[string]any {
	if contribution == nil {
		return nil
	}
	return map[string]any{
		"id":          contribution.Id,
		"title":       contribution.Title,
		"category":    contribution.Category,
		"path":        contribution.Path,
		"render_mode": contribution.RenderMode,
		"app_context": contribution.AppContext,
		"actions":     append([]string(nil), contribution.Actions...),
	}
}

func authorizeFromEvidence(args map[string]any) (bool, string) {
	if checked, _ := args["authz_checked"].(bool); !checked {
		return false, "missing authz evidence"
	}
	if allowed, _ := args["authz_allowed"].(bool); !allowed {
		return false, "authz denied"
	}
	return true, "authz allowed"
}

func stringValue(args map[string]any, key string) string {
	value, _ := args[key].(string)
	return value
}

func stringSliceValue(args map[string]any, key string) []string {
	switch value := args[key].(type) {
	case []string:
		return append([]string(nil), value...)
	case []any:
		out := make([]string, 0, len(value))
		for _, item := range value {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
