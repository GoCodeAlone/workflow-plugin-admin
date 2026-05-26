package internal

import (
	"testing"

	"github.com/GoCodeAlone/workflow/plugin/external/proto"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func TestAdminPlugin_ContractRegistry(t *testing.T) {
	provider, ok := NewAdminPlugin().(sdk.ContractProvider)
	if !ok {
		t.Fatal("expected admin plugin to implement sdk.ContractProvider")
	}

	registry := provider.ContractRegistry()
	if registry == nil {
		t.Fatal("expected non-nil contract registry")
	}
	if registry.FileDescriptorSet == nil || len(registry.FileDescriptorSet.File) == 0 {
		t.Fatal("expected contract registry to include file descriptors")
	}

	contracts := make(map[string]*proto.ContractDescriptor, len(registry.Contracts))
	for _, contract := range registry.Contracts {
		if contract.Mode != proto.ContractMode_CONTRACT_MODE_STRICT_PROTO {
			t.Fatalf("%s mode = %s, want strict proto", contractKey(contract), contract.Mode)
		}
		key := contractKey(contract)
		if _, exists := contracts[key]; exists {
			t.Fatalf("duplicate contract %q", key)
		}
		contracts[key] = contract
	}

	requireContract(t, contracts, "module:admin.dashboard", "workflow.plugins.admin.v1.AdminDashboardConfig", "", "")
	requireContract(t, contracts, "step:step.admin_register_contribution", "workflow.plugins.admin.v1.AdminStepConfig", "workflow.plugins.admin.v1.RegisterContributionInput", "workflow.plugins.admin.v1.RegisterContributionOutput")
	requireContract(t, contracts, "step:step.admin_list_contributions", "workflow.plugins.admin.v1.AdminStepConfig", "workflow.plugins.admin.v1.ListContributionsInput", "workflow.plugins.admin.v1.ListContributionsOutput")
	requireContract(t, contracts, "step:step.admin_authorize_action", "workflow.plugins.admin.v1.AdminStepConfig", "workflow.plugins.admin.v1.AuthorizeAdminActionInput", "workflow.plugins.admin.v1.AuthorizeAdminActionOutput")
	requireContract(t, contracts, "step:step.admin_resource_action", "workflow.plugins.admin.v1.AdminStepConfig", "workflow.plugins.admin.v1.AdminResourceActionInput", "workflow.plugins.admin.v1.AdminResourceActionOutput")
}

func contractKey(contract *proto.ContractDescriptor) string {
	switch contract.Kind {
	case proto.ContractKind_CONTRACT_KIND_MODULE:
		return "module:" + contract.ModuleType
	case proto.ContractKind_CONTRACT_KIND_STEP:
		return "step:" + contract.StepType
	case proto.ContractKind_CONTRACT_KIND_SERVICE:
		return "service:" + contract.ModuleType + "/" + contract.ServiceName + "/" + contract.Method
	case proto.ContractKind_CONTRACT_KIND_TRIGGER:
		return "trigger:" + contract.TriggerType
	default:
		return "unknown"
	}
}

func requireContract(t *testing.T, contracts map[string]*proto.ContractDescriptor, key, config, input, output string) {
	t.Helper()
	contract, ok := contracts[key]
	if !ok {
		t.Fatalf("missing contract %q", key)
	}
	if contract.ConfigMessage != config {
		t.Fatalf("%s config = %q, want %q", key, contract.ConfigMessage, config)
	}
	if contract.InputMessage != input {
		t.Fatalf("%s input = %q, want %q", key, contract.InputMessage, input)
	}
	if contract.OutputMessage != output {
		t.Fatalf("%s output = %q, want %q", key, contract.OutputMessage, output)
	}
}
