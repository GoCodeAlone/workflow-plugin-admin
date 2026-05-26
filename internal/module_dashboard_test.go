package internal

import (
	"context"
	"testing"

	"github.com/GoCodeAlone/workflow-plugin-admin/internal/contracts"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func TestContributionRegistryValidatesAndSorts(t *testing.T) {
	reg := newContributionRegistry()

	if err := reg.register(&contracts.AdminContribution{Id: "bad id", Title: "Bad", Path: "/bad"}); err == nil {
		t.Fatal("expected invalid id to be rejected")
	}
	if err := reg.register(&contracts.AdminContribution{Id: "good", Title: "Bad Path", Path: "admin/good"}); err == nil {
		t.Fatal("expected path without leading slash to be rejected")
	}

	for _, contribution := range []*contracts.AdminContribution{
		{Id: "zeta", Title: "Zeta", Path: "/admin/zeta", Category: "tools"},
		{Id: "alpha", Title: "Alpha", Path: "/admin/alpha", Category: "tools"},
		{Id: "zeta", Title: "Zeta Updated", Path: "/admin/zeta", Category: "tools"},
	} {
		if err := reg.register(contribution); err != nil {
			t.Fatalf("register %s: %v", contribution.Id, err)
		}
	}

	list := reg.list("")
	if len(list) != 2 {
		t.Fatalf("len(list) = %d, want 2", len(list))
	}
	if list[0].Id != "alpha" || list[1].Id != "zeta" {
		t.Fatalf("list order = [%s %s], want [alpha zeta]", list[0].Id, list[1].Id)
	}
	if list[1].Title != "Zeta Updated" {
		t.Fatalf("duplicate id did not update contribution: %q", list[1].Title)
	}
}

func TestDashboardModuleServiceInvoker(t *testing.T) {
	module := newDashboardModule("admin", &contracts.AdminDashboardConfig{
		RoutePrefix: "/admin",
		AppName:     "Scenario App",
		TargetApp:   "scenario",
	})

	if err := module.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	invoker, ok := any(module).(sdk.ServiceInvoker)
	if !ok {
		t.Fatalf("dashboard module should implement ServiceInvoker")
	}

	registered, err := invoker.InvokeMethod("RegisterContribution", map[string]any{
		"id":          "orders",
		"title":       "Orders",
		"path":        "/admin/orders",
		"category":    "operations",
		"render_mode": "json-schema",
		"app_context": "scenario",
	})
	if err != nil {
		t.Fatalf("RegisterContribution: %v", err)
	}
	if registered["registered"] != true {
		t.Fatalf("registered = %v, want true", registered["registered"])
	}

	listed, err := invoker.InvokeMethod("ListContributions", map[string]any{"app_context": "scenario"})
	if err != nil {
		t.Fatalf("ListContributions: %v", err)
	}
	contributions, ok := listed["contributions"].([]map[string]any)
	if !ok {
		t.Fatalf("contributions type = %T, want []map[string]any", listed["contributions"])
	}
	if len(contributions) != 1 || contributions[0]["id"] != "orders" {
		t.Fatalf("unexpected contributions: %#v", contributions)
	}
}

func TestDashboardModuleAuthorizeDefaultsDeny(t *testing.T) {
	module := newDashboardModule("admin", &contracts.AdminDashboardConfig{})
	invoker := any(module).(sdk.ServiceInvoker)

	result, err := invoker.InvokeMethod("AuthorizeAction", map[string]any{
		"subject":  "alice",
		"resource": "admin:users",
		"action":   "write",
	})
	if err != nil {
		t.Fatalf("AuthorizeAction: %v", err)
	}
	if result["allowed"] != false {
		t.Fatalf("allowed = %v, want false", result["allowed"])
	}
	if result["reason"] == "" {
		t.Fatal("expected deny reason")
	}
}

func TestAdminPluginProvidesDashboardModule(t *testing.T) {
	plugin := NewAdminPlugin()
	provider, ok := plugin.(sdk.ModuleProvider)
	if !ok {
		t.Fatal("expected admin plugin to implement ModuleProvider")
	}
	if _, ok := plugin.(sdk.TypedModuleProvider); !ok {
		t.Fatal("expected admin plugin to implement TypedModuleProvider")
	}

	if got := provider.ModuleTypes(); len(got) != 1 || got[0] != "admin.dashboard" {
		t.Fatalf("ModuleTypes() = %#v, want [admin.dashboard]", got)
	}
	module, err := provider.CreateModule("admin.dashboard", "admin", map[string]any{"route_prefix": "/admin"})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}
	if _, ok := module.(sdk.ServiceInvoker); !ok {
		t.Fatalf("module %T does not implement ServiceInvoker", module)
	}
}
