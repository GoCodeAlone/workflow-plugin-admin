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
		request := contributionListRequest{
			AppContext:          stringValue(args, "app_context"),
			SelectedContextKind: stringValue(args, "selected_context_kind"),
			SelectedContextID:   stringValue(args, "selected_context_id"),
			ContextAuthorized:   boolValue(args, "context_authorized"),
			GrantedPermissions:  stringSliceValue(args, "granted_permissions"),
		}
		contributions := contributionListValue(args["contributions"])
		if len(contributions) == 0 {
			contributions = contributionListValue([]any{args["auth_contribution"], args["authz_contribution"]})
		}
		if len(contributions) == 0 {
			contributions = m.registry.listForRequest(request)
		} else {
			contributions = filterContributionList(contributions, request)
		}
		out := make([]any, 0, len(contributions))
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
	if selector := contextSelectorValue(args["context_selector"]); selector != nil {
		contribution.ContextSelector = selector
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

func contributionListValue(value any) []*contracts.AdminContribution {
	switch items := value.(type) {
	case []*contracts.AdminContribution:
		return items
	case []any:
		out := make([]*contracts.AdminContribution, 0, len(items))
		for _, item := range items {
			switch contribution := item.(type) {
			case *contracts.AdminContribution:
				out = append(out, contribution)
			case map[string]any:
				out = append(out, contributionFromMap(contribution))
			}
		}
		return out
	default:
		return nil
	}
}

func contributionToMap(contribution *contracts.AdminContribution) map[string]any {
	if contribution == nil {
		return nil
	}
	out := map[string]any{
		"id":          contribution.Id,
		"title":       contribution.Title,
		"category":    contribution.Category,
		"path":        contribution.Path,
		"render_mode": contribution.RenderMode,
		"app_context": contribution.AppContext,
		"actions":     stringAnySlice(contribution.Actions),
		"permissions": permissionMaps(contribution.Permissions),
		"metadata":    contributionMetadataMap(contribution),
	}
	if contribution.ContextSelector != nil {
		out["context_selector"] = contextSelectorMap(contribution.ContextSelector)
	}
	return out
}

func stringAnySlice(values []string) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
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
			case map[any]any:
				permissionMap := stringMapValue(permission)
				out = append(out, &contracts.AdminPermission{
					Permission: stringValue(permissionMap, "permission"),
					Resource:   stringValue(permissionMap, "resource"),
					Action:     stringValue(permissionMap, "action"),
				})
			}
		}
		return out
	default:
		return nil
	}
}

func permissionMaps(permissions []*contracts.AdminPermission) []any {
	out := make([]any, 0, len(permissions))
	for _, permission := range permissions {
		out = append(out, map[string]any{
			"permission": permission.GetPermission(),
			"resource":   permission.GetResource(),
			"action":     permission.GetAction(),
		})
	}
	return out
}

func contextSelectorValue(value any) *contracts.AdminContextSelector {
	selectorMap := stringMapValue(value)
	if selectorMap == nil {
		return nil
	}
	return &contracts.AdminContextSelector{
		SelectedContextKey:  stringValue(selectorMap, "selected_context_key"),
		AllowedContextKinds: stringSliceValue(selectorMap, "allowed_context_kinds"),
		LaunchUrl:           stringValue(selectorMap, "launch_url"),
		SwitchPermissions:   permissionValues(selectorMap["switch_permissions"]),
	}
}

func contextSelectorMap(selector *contracts.AdminContextSelector) map[string]any {
	if selector == nil {
		return nil
	}
	return map[string]any{
		"selected_context_key":  selector.GetSelectedContextKey(),
		"allowed_context_kinds": stringAnySlice(selector.GetAllowedContextKinds()),
		"launch_url":            selector.GetLaunchUrl(),
		"switch_permissions":    permissionMaps(selector.GetSwitchPermissions()),
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

func boolValue(args map[string]any, key string) bool {
	value, _ := args[key].(bool)
	return value
}

func stringValue(args map[string]any, key string) string {
	value, _ := args[key].(string)
	return value
}

func mapValue(args map[string]any, key string) map[string]any {
	return stringMapValue(args[key])
}

func stringMapValue(value any) map[string]any {
	switch value := value.(type) {
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
