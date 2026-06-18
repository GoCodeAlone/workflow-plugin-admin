package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoCodeAlone/workflow-plugin-admin/internal/contracts"
	"github.com/GoCodeAlone/workflow/plugin/external/proto"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
	"google.golang.org/protobuf/reflect/protoreflect"
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
		if contractKey(contract) == "step:step.admin_register_contribution" || contractKey(contract) == "step:step.admin_list_contributions" {
			if contract.Mode != proto.ContractMode_CONTRACT_MODE_PROTO_WITH_LEGACY_STRUCT {
				t.Fatalf("%s mode = %s, want proto with legacy struct", contractKey(contract), contract.Mode)
			}
		} else if contract.Mode != proto.ContractMode_CONTRACT_MODE_STRICT_PROTO {
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

func TestAdminPlugin_ContextSelectorStrictProtoFields(t *testing.T) {
	messages := contracts.File_internal_contracts_admin_proto.Messages()

	selector := messages.ByName("AdminContextSelector")
	if selector == nil {
		t.Fatal("AdminContextSelector missing from strict proto descriptor")
	}
	requireField(t, selector, "selected_context_key")
	requireField(t, selector, "allowed_context_kinds")
	requireField(t, selector, "launch_url")
	requireField(t, selector, "switch_permissions")

	contribution := messages.ByName("AdminContribution")
	if contribution == nil {
		t.Fatal("AdminContribution missing from strict proto descriptor")
	}
	requireField(t, contribution, "context_selector")

	listInput := messages.ByName("ListContributionsInput")
	if listInput == nil {
		t.Fatal("ListContributionsInput missing from strict proto descriptor")
	}
	requireField(t, listInput, "selected_context_kind")
	requireField(t, listInput, "selected_context_id")
	requireField(t, listInput, "context_authorized")
}

func TestAdminPlugin_PluginContractsJSON(t *testing.T) {
	provider := NewAdminPlugin().(sdk.ContractProvider)
	runtimeContracts := make(map[string]*proto.ContractDescriptor)
	for _, contract := range provider.ContractRegistry().Contracts {
		runtimeContracts[contractKey(contract)] = contract
	}

	manifestContracts := readPluginContracts(t)
	if len(manifestContracts) != len(runtimeContracts) {
		t.Fatalf("manifest contract count = %d, runtime = %d", len(manifestContracts), len(runtimeContracts))
	}
	for key, manifest := range manifestContracts {
		runtimeContract, ok := runtimeContracts[key]
		if !ok {
			t.Fatalf("%s missing from runtime contracts", key)
		}
		if manifest.Config != runtimeContract.ConfigMessage || manifest.Input != runtimeContract.InputMessage || manifest.Output != runtimeContract.OutputMessage || manifest.Mode != contractModeString(runtimeContract.Mode) {
			t.Fatalf("%s manifest = %#v, runtime mode/config/input/output = %q/%q/%q/%q", key, manifest, contractModeString(runtimeContract.Mode), runtimeContract.ConfigMessage, runtimeContract.InputMessage, runtimeContract.OutputMessage)
		}
	}
}

func requireField(t *testing.T, message protoreflect.MessageDescriptor, name protoreflect.Name) {
	t.Helper()
	if message.Fields().ByName(name) == nil {
		t.Fatalf("%s missing field %s", message.FullName(), name)
	}
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

type pluginContract struct {
	Config string `json:"config"`
	Input  string `json:"input"`
	Output string `json:"output"`
	Mode   string `json:"mode"`
}

func contractModeString(mode proto.ContractMode) string {
	switch mode {
	case proto.ContractMode_CONTRACT_MODE_STRICT_PROTO:
		return "strict"
	case proto.ContractMode_CONTRACT_MODE_PROTO_WITH_LEGACY_STRUCT:
		return "proto_with_legacy_struct"
	case proto.ContractMode_CONTRACT_MODE_LEGACY_STRUCT:
		return "legacy_struct"
	default:
		return ""
	}
}

func readPluginContracts(t *testing.T) map[string]pluginContract {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "plugin.contracts.json"))
	if err != nil {
		t.Fatalf("read plugin.contracts.json: %v", err)
	}
	var manifest struct {
		Version   string `json:"version"`
		Contracts []struct {
			Kind        string `json:"kind"`
			Type        string `json:"type"`
			ServiceName string `json:"serviceName"`
			Method      string `json:"method"`
			pluginContract
		} `json:"contracts"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse plugin.contracts.json: %v", err)
	}
	if manifest.Version != "v1" {
		t.Fatalf("plugin.contracts.json version = %q, want v1", manifest.Version)
	}
	out := make(map[string]pluginContract, len(manifest.Contracts))
	for _, contract := range manifest.Contracts {
		var key string
		switch contract.Kind {
		case "module":
			key = "module:" + contract.Type
		case "step":
			key = "step:" + contract.Type
		case "service_method":
			key = "service:" + contract.Type + "/" + contract.ServiceName + "/" + contract.Method
		default:
			t.Fatalf("unexpected contract kind %q", contract.Kind)
		}
		if _, exists := out[key]; exists {
			t.Fatalf("duplicate manifest contract %q", key)
		}
		out[key] = contract.pluginContract
	}
	return out
}
