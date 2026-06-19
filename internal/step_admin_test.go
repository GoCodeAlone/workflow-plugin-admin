package internal

import (
	"context"
	"errors"
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
	contributions, ok := result.Output["contributions"].([]any)
	if !ok {
		t.Fatalf("contributions type = %T", result.Output["contributions"])
	}
	first, ok := contributions[0].(map[string]any)
	if !ok {
		t.Fatalf("first contribution type = %T, want map[string]any", contributions[0])
	}
	if len(contributions) != 1 || first["id"] != "orders" {
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

func TestContributionRegistryStepsUseLegacyExecutionWithTypedContractDescriptors(t *testing.T) {
	provider := NewAdminPlugin().(sdk.TypedStepProvider)

	goodConfig, err := anypb.New(&contracts.AdminStepConfig{Module: "admin"})
	if err != nil {
		t.Fatalf("pack good config: %v", err)
	}
	if _, err := provider.CreateTypedStep("step.admin_register_contribution", "register", goodConfig); !errors.Is(err, sdk.ErrTypedContractNotHandled) {
		t.Fatalf("CreateTypedStep register error = %v, want ErrTypedContractNotHandled", err)
	}
	if _, err := provider.CreateTypedStep("step.admin_list_contributions", "list", goodConfig); !errors.Is(err, sdk.ErrTypedContractNotHandled) {
		t.Fatalf("CreateTypedStep list error = %v, want ErrTypedContractNotHandled", err)
	}

	steps := NewAdminPlugin().(sdk.TypedStepProvider).TypedStepTypes()
	for _, stepType := range steps {
		if stepType == "step.admin_register_contribution" || stepType == "step.admin_list_contributions" {
			t.Fatalf("%s must use the legacy StepInstance path for map-shaped contribution arrays", stepType)
		}
	}
}

func TestTypedAdminStepsUseCurrentFallbackForWorkflowInputs(t *testing.T) {
	module := newDashboardModule("admin", &contracts.AdminDashboardConfig{})
	registerDashboardModule(module)

	registerResult, err := typedRegisterContribution(context.Background(), sdk.TypedStepRequest[*contracts.AdminStepConfig, *contracts.RegisterContributionInput]{
		Config: &contracts.AdminStepConfig{Module: "admin"},
		Input:  &contracts.RegisterContributionInput{},
		Current: map[string]any{
			"contribution": map[string]any{
				"id":          "authz-roles",
				"title":       "Authorization",
				"path":        "/admin/authz/",
				"category":    "security",
				"render_mode": "iframe",
				"app_context": "admin",
				"permissions": []any{"admin:authz.roles:read"},
			},
		},
	})
	if err != nil {
		t.Fatalf("typedRegisterContribution: %v", err)
	}
	if !registerResult.Output.GetRegistered() {
		t.Fatal("typed register did not report registered")
	}

	listResult, err := typedListContributions(context.Background(), sdk.TypedStepRequest[*contracts.AdminStepConfig, *contracts.ListContributionsInput]{
		Config: &contracts.AdminStepConfig{Module: "admin"},
		Input:  &contracts.ListContributionsInput{},
		Current: map[string]any{
			"app_context":         "admin",
			"granted_permissions": []any{"admin:authz.roles:read"},
		},
	})
	if err != nil {
		t.Fatalf("typedListContributions: %v", err)
	}
	if len(listResult.Output.GetContributions()) != 1 || listResult.Output.GetContributions()[0].GetId() != "authz-roles" {
		t.Fatalf("unexpected typed list output: %#v", listResult.Output.GetContributions())
	}
}

func TestTypedListContributionsCarriesTrustedContextEvidence(t *testing.T) {
	module := newDashboardModule("admin", &contracts.AdminDashboardConfig{})
	registerDashboardModule(module)

	registerResult, err := typedRegisterContribution(context.Background(), sdk.TypedStepRequest[*contracts.AdminStepConfig, *contracts.RegisterContributionInput]{
		Config: &contracts.AdminStepConfig{Module: "admin"},
		Input: &contracts.RegisterContributionInput{
			Contribution: &contracts.AdminContribution{
				Id:    "cms-site",
				Title: "CMS Site",
				Path:  "/admin/sites",
				Permissions: []*contracts.AdminPermission{
					{Permission: "admin:cms.site:read"},
				},
				ContextSelector: &contracts.AdminContextSelector{
					SelectedContextKey:  "site",
					AllowedContextKinds: []string{"site"},
					SwitchPermissions: []*contracts.AdminPermission{
						{Permission: "admin:cms.site:switch"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("typedRegisterContribution: %v", err)
	}
	if !registerResult.Output.GetRegistered() {
		t.Fatal("typed register did not report registered")
	}

	listResult, err := typedListContributions(context.Background(), sdk.TypedStepRequest[*contracts.AdminStepConfig, *contracts.ListContributionsInput]{
		Config: &contracts.AdminStepConfig{Module: "admin"},
		Input: &contracts.ListContributionsInput{
			SelectedContextKind: "site",
			SelectedContextId:   "blackorchid",
			ContextAuthorized:   true,
			GrantedPermissions:  []string{"admin:cms.site:read", "admin:cms.site:switch"},
		},
	})
	if err != nil {
		t.Fatalf("typedListContributions: %v", err)
	}
	if len(listResult.Output.GetContributions()) != 1 || listResult.Output.GetContributions()[0].GetId() != "cms-site" {
		t.Fatalf("unexpected typed context list output: %#v", listResult.Output.GetContributions())
	}
}
