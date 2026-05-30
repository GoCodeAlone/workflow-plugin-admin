package internal

import (
	"context"
	"fmt"
	"sync"

	"github.com/GoCodeAlone/workflow-plugin-admin/internal/contracts"
	"google.golang.org/protobuf/types/known/structpb"
)

var dashboardModules = struct {
	sync.RWMutex
	byName map[string]*dashboardModule
}{byName: make(map[string]*dashboardModule)}

type dashboardModule struct {
	name     string
	config   *contracts.AdminDashboardConfig
	registry *contributionRegistry
}

func registerDashboardModule(module *dashboardModule) {
	dashboardModules.Lock()
	defer dashboardModules.Unlock()
	dashboardModules.byName[module.name] = module
}

func lookupDashboardModule(name string) (*dashboardModule, error) {
	if name == "" {
		name = "admin"
	}
	dashboardModules.RLock()
	defer dashboardModules.RUnlock()
	module, ok := dashboardModules.byName[name]
	if !ok {
		return nil, fmt.Errorf("admin dashboard module %q not found", name)
	}
	return module, nil
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
		contributions := m.registry.listForPermissions(appContext, stringSliceValue(args, "granted_permissions"))
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
	contribution := &contracts.AdminContribution{
		Id:         stringValue(args, "id"),
		Title:      stringValue(args, "title"),
		Category:   stringValue(args, "category"),
		Path:       stringValue(args, "path"),
		RenderMode: stringValue(args, "render_mode"),
		AppContext: stringValue(args, "app_context"),
		Actions:    stringSliceValue(args, "actions"),
	}
	if metadata := mapValue(args, "metadata"); len(metadata) > 0 {
		if pbMetadata, err := structpb.NewStruct(metadata); err == nil {
			contribution.Metadata = pbMetadata
		}
	}
	for _, permission := range permissionValues(args["permissions"]) {
		contribution.Permissions = append(contribution.Permissions, permission)
	}
	return contribution
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
		"permissions": permissionMaps(contribution.Permissions),
		"metadata":    contributionMetadataMap(contribution),
	}
}

func contributionMetadataMap(contribution *contracts.AdminContribution) map[string]any {
	if contribution == nil || contribution.Metadata == nil {
		return map[string]any{}
	}
	return contribution.Metadata.AsMap()
}

func permissionValues(value any) []*contracts.AdminPermission {
	switch permissions := value.(type) {
	case []string:
		out := make([]*contracts.AdminPermission, 0, len(permissions))
		for _, permission := range permissions {
			out = append(out, &contracts.AdminPermission{Permission: permission})
		}
		return out
	case []any:
		out := make([]*contracts.AdminPermission, 0, len(permissions))
		for _, item := range permissions {
			switch permission := item.(type) {
			case string:
				out = append(out, &contracts.AdminPermission{Permission: permission})
			case map[string]any:
				out = append(out, &contracts.AdminPermission{
					Permission: stringValue(permission, "permission"),
					Resource:   stringValue(permission, "resource"),
					Action:     stringValue(permission, "action"),
				})
			}
		}
		return out
	default:
		return nil
	}
}

func permissionMaps(permissions []*contracts.AdminPermission) []map[string]any {
	out := make([]map[string]any, 0, len(permissions))
	for _, permission := range permissions {
		out = append(out, map[string]any{
			"permission": permission.GetPermission(),
			"resource":   permission.GetResource(),
			"action":     permission.GetAction(),
		})
	}
	return out
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

func mapValue(args map[string]any, key string) map[string]any {
	switch value := args[key].(type) {
	case map[string]any:
		return value
	case map[any]any:
		out := make(map[string]any, len(value))
		for k, v := range value {
			if s, ok := k.(string); ok {
				out[s] = v
			}
		}
		return out
	default:
		return nil
	}
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
