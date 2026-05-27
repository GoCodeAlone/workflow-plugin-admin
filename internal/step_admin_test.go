package internal

import (
	"context"
	"testing"

	"github.com/GoCodeAlone/workflow-plugin-admin/internal/contracts"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestAdminStepsRegisterListAuthorizeAndDispatch(t *testing.T) {
	plugin := NewAdminPlugin().(interface {
		sdk.ModuleProvider
		sdk.StepProvider
	})
	module, err := plugin.CreateModule("admin.dashboard", "admin", nil)
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}
	if err := module.Init(); err != nil {
		t.Fatalf("module Init: %v", err)
	}

	register, err := plugin.CreateStep("step.admin_register_contribution", "register", map[string]any{"module": "admin"})
	if err != nil {
		t.Fatalf("CreateStep register: %v", err)
	}
	result, err := register.Execute(context.Background(), nil, nil, map[string]any{
		"id":          "orders",
		"title":       "Orders",
		"path":        "/admin/orders",
		"category":    "operations",
		"render_mode": "json-schema",
	}, nil, nil)
	if err != nil {
		t.Fatalf("register Execute: %v", err)
	}
	if result.Output["registered"] != true {
		t.Fatalf("registered = %v, want true", result.Output["registered"])
	}

	list, err := plugin.CreateStep("step.admin_list_contributions", "list", map[string]any{"module": "admin"})
	if err != nil {
		t.Fatalf("CreateStep list: %v", err)
	}
	result, err = list.Execute(context.Background(), nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("list Execute: %v", err)
	}
	contributions, ok := result.Output["contributions"].([]map[string]any)
	if !ok {
		t.Fatalf("contributions type = %T", result.Output["contributions"])
	}
	if len(contributions) != 1 || contributions[0]["id"] != "orders" {
		t.Fatalf("unexpected contributions: %#v", contributions)
	}

	authorize, err := plugin.CreateStep("step.admin_authorize_action", "authorize", map[string]any{"module": "admin"})
	if err != nil {
		t.Fatalf("CreateStep authorize: %v", err)
	}
	result, err = authorize.Execute(context.Background(), nil, nil, map[string]any{"resource": "admin:orders", "action": "read"}, nil, nil)
	if err != nil {
		t.Fatalf("authorize Execute: %v", err)
	}
	if result.Output["allowed"] != false {
		t.Fatalf("allowed without evidence = %v, want false", result.Output["allowed"])
	}

	dispatch, err := plugin.CreateStep("step.admin_resource_action", "dispatch", map[string]any{"module": "admin"})
	if err != nil {
		t.Fatalf("CreateStep dispatch: %v", err)
	}
	result, err = dispatch.Execute(context.Background(), nil, nil, map[string]any{
		"contribution_id": "orders",
		"resource":        "admin:orders",
		"action":          "read",
		"authz_checked":   true,
		"authz_allowed":   true,
	}, nil, nil)
	if err != nil {
		t.Fatalf("dispatch Execute: %v", err)
	}
	actionResult, ok := result.Output["result"].(map[string]any)
	if !ok {
		t.Fatalf("result type = %T", result.Output["result"])
	}
	if actionResult["accepted"] != true {
		t.Fatalf("accepted = %v, want true", actionResult["accepted"])
	}
}

func TestAdminTypedStepProviderValidatesConfig(t *testing.T) {
	provider := NewAdminPlugin().(sdk.TypedStepProvider)

	goodConfig, err := anypb.New(&contracts.AdminStepConfig{Module: "admin"})
	if err != nil {
		t.Fatalf("pack good config: %v", err)
	}
	if _, err := provider.CreateTypedStep("step.admin_list_contributions", "list", goodConfig); err != nil {
		t.Fatalf("CreateTypedStep good config: %v", err)
	}

	wrongConfig, err := anypb.New(&contracts.AdminDashboardConfig{RoutePrefix: "/admin"})
	if err != nil {
		t.Fatalf("pack wrong config: %v", err)
	}
	if _, err := provider.CreateTypedStep("step.admin_list_contributions", "list", wrongConfig); err == nil {
		t.Fatal("CreateTypedStep accepted wrong config type")
	}
}
