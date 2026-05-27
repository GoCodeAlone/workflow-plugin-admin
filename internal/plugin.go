package internal

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoCodeAlone/workflow-plugin-admin/internal/contracts"
	pb "github.com/GoCodeAlone/workflow/plugin/external/proto"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v3"
)

// Version is set at build time via -ldflags
// "-X github.com/GoCodeAlone/workflow-plugin-admin/internal.Version=X.Y.Z".
// Default is a bare semver so plugin loaders that validate semver accept
// unreleased dev builds; goreleaser overrides with the real release tag.
var Version = "0.0.0"

// adminPlugin implements PluginProvider and ConfigProvider.
type adminPlugin struct{}

var adminStepTypes = []string{
	"step.admin_register_contribution",
	"step.admin_list_contributions",
	"step.admin_authorize_action",
	"step.admin_resource_action",
}

// NewAdminPlugin returns a new adminPlugin instance.
func NewAdminPlugin() sdk.PluginProvider {
	return &adminPlugin{}
}

// Manifest returns plugin metadata.
func (p *adminPlugin) Manifest() sdk.PluginManifest {
	return sdk.PluginManifest{
		Name:        "admin",
		Version:     Version,
		Author:      "GoCodeAlone",
		Description: "Admin dashboard UI and config-driven admin routes",
	}
}

// ConfigFragment returns the embedded admin config.yaml with absolute paths
// resolved relative to the plugin's working directory.
func (p *adminPlugin) ConfigFragment() ([]byte, error) {
	if err := extractAssets(); err != nil {
		return nil, fmt.Errorf("admin plugin: extract assets: %w", err)
	}

	dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("admin plugin: get working directory: %w", err)
	}

	absUIPath := filepath.Join(dir, "ui_dist")

	var cfg map[string]any
	if err := yaml.Unmarshal(configData, &cfg); err != nil {
		return nil, fmt.Errorf("admin plugin: parse config: %w", err)
	}

	// Update static.fileserver and admin.dashboard root paths to absolute paths.
	if modules, ok := cfg["modules"].([]any); ok {
		for _, m := range modules {
			mod, ok := m.(map[string]any)
			if !ok {
				continue
			}
			modType, _ := mod["type"].(string)
			if modType == "static.fileserver" || modType == "admin.dashboard" {
				if config, ok := mod["config"].(map[string]any); ok {
					config["root"] = absUIPath
				}
			}
		}
	}

	return yaml.Marshal(cfg)
}

func (p *adminPlugin) ModuleTypes() []string {
	return []string{"admin.dashboard"}
}

func (p *adminPlugin) TypedModuleTypes() []string {
	return p.ModuleTypes()
}

func (p *adminPlugin) CreateModule(typeName, name string, config map[string]any) (sdk.ModuleInstance, error) {
	if typeName != "admin.dashboard" {
		return nil, fmt.Errorf("admin plugin: unknown module type %q", typeName)
	}
	module := newDashboardModule(name, dashboardConfigFromMap(config))
	registerDashboardModule(module)
	return module, nil
}

func (p *adminPlugin) CreateTypedModule(typeName, name string, config *anypb.Any) (sdk.ModuleInstance, error) {
	if typeName != "admin.dashboard" {
		return nil, fmt.Errorf("admin plugin: unknown typed module type %q", typeName)
	}
	cfg := &contracts.AdminDashboardConfig{}
	if config != nil {
		if err := config.UnmarshalTo(cfg); err != nil {
			return nil, fmt.Errorf("admin dashboard config: %w", err)
		}
	}
	module := newDashboardModule(name, cfg)
	registerDashboardModule(module)
	return module, nil
}

func (p *adminPlugin) StepTypes() []string {
	return append([]string(nil), adminStepTypes...)
}

func (p *adminPlugin) TypedStepTypes() []string {
	return p.StepTypes()
}

func (p *adminPlugin) CreateStep(typeName, _ string, config map[string]any) (sdk.StepInstance, error) {
	switch typeName {
	case "step.admin_register_contribution":
		return newAdminStep("RegisterContribution", config), nil
	case "step.admin_list_contributions":
		return newAdminStep("ListContributions", config), nil
	case "step.admin_authorize_action":
		return newAdminStep("AuthorizeAction", config), nil
	case "step.admin_resource_action":
		return newAdminStep("DispatchResourceAction", config), nil
	default:
		return nil, fmt.Errorf("admin plugin: unknown step type %q", typeName)
	}
}

func (p *adminPlugin) CreateTypedStep(typeName, name string, config *anypb.Any) (sdk.StepInstance, error) {
	switch typeName {
	case "step.admin_register_contribution":
		return sdk.NewTypedStepFactory(typeName, &contracts.AdminStepConfig{}, &contracts.RegisterContributionInput{}, typedRegisterContribution).CreateTypedStep(typeName, name, config)
	case "step.admin_list_contributions":
		return sdk.NewTypedStepFactory(typeName, &contracts.AdminStepConfig{}, &contracts.ListContributionsInput{}, typedListContributions).CreateTypedStep(typeName, name, config)
	case "step.admin_authorize_action":
		return sdk.NewTypedStepFactory(typeName, &contracts.AdminStepConfig{}, &contracts.AuthorizeAdminActionInput{}, typedAuthorizeAction).CreateTypedStep(typeName, name, config)
	case "step.admin_resource_action":
		return sdk.NewTypedStepFactory(typeName, &contracts.AdminStepConfig{}, &contracts.AdminResourceActionInput{}, typedResourceAction).CreateTypedStep(typeName, name, config)
	default:
		return nil, fmt.Errorf("admin plugin: unknown typed step type %q", typeName)
	}
}

// ContractRegistry returns the admin plugin's strict protobuf contracts.
func (p *adminPlugin) ContractRegistry() *pb.ContractRegistry {
	return &pb.ContractRegistry{
		FileDescriptorSet: &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{
			protodesc.ToFileDescriptorProto(structpb.File_google_protobuf_struct_proto),
			protodesc.ToFileDescriptorProto(contracts.File_internal_contracts_admin_proto),
		}},
		Contracts: []*pb.ContractDescriptor{
			adminModuleContract("admin.dashboard", "AdminDashboardConfig"),
			adminStepContract("step.admin_register_contribution", "AdminStepConfig", "RegisterContributionInput", "RegisterContributionOutput"),
			adminStepContract("step.admin_list_contributions", "AdminStepConfig", "ListContributionsInput", "ListContributionsOutput"),
			adminStepContract("step.admin_authorize_action", "AdminStepConfig", "AuthorizeAdminActionInput", "AuthorizeAdminActionOutput"),
			adminStepContract("step.admin_resource_action", "AdminStepConfig", "AdminResourceActionInput", "AdminResourceActionOutput"),
			adminServiceContract("admin.dashboard", "AdminDashboard", "RegisterContribution", "RegisterContributionInput", "RegisterContributionOutput"),
			adminServiceContract("admin.dashboard", "AdminDashboard", "ListContributions", "ListContributionsInput", "ListContributionsOutput"),
			adminServiceContract("admin.dashboard", "AdminDashboard", "AuthorizeAction", "AuthorizeAdminActionInput", "AuthorizeAdminActionOutput"),
			adminServiceContract("admin.dashboard", "AdminDashboard", "DispatchResourceAction", "AdminResourceActionInput", "AdminResourceActionOutput"),
		},
	}
}

func adminModuleContract(moduleType, configMessage string) *pb.ContractDescriptor {
	const pkg = "workflow.plugins.admin.v1."
	return &pb.ContractDescriptor{
		Kind:          pb.ContractKind_CONTRACT_KIND_MODULE,
		ModuleType:    moduleType,
		ConfigMessage: pkg + configMessage,
		Mode:          pb.ContractMode_CONTRACT_MODE_STRICT_PROTO,
	}
}

func adminStepContract(stepType, configMessage, inputMessage, outputMessage string) *pb.ContractDescriptor {
	const pkg = "workflow.plugins.admin.v1."
	return &pb.ContractDescriptor{
		Kind:          pb.ContractKind_CONTRACT_KIND_STEP,
		StepType:      stepType,
		ConfigMessage: pkg + configMessage,
		InputMessage:  pkg + inputMessage,
		OutputMessage: pkg + outputMessage,
		Mode:          pb.ContractMode_CONTRACT_MODE_STRICT_PROTO,
	}
}

func adminServiceContract(moduleType, serviceName, method, inputMessage, outputMessage string) *pb.ContractDescriptor {
	const pkg = "workflow.plugins.admin.v1."
	return &pb.ContractDescriptor{
		Kind:          pb.ContractKind_CONTRACT_KIND_SERVICE,
		ModuleType:    moduleType,
		ServiceName:   serviceName,
		Method:        method,
		InputMessage:  pkg + inputMessage,
		OutputMessage: pkg + outputMessage,
		Mode:          pb.ContractMode_CONTRACT_MODE_STRICT_PROTO,
	}
}

func dashboardConfigFromMap(config map[string]any) *contracts.AdminDashboardConfig {
	return &contracts.AdminDashboardConfig{
		RoutePrefix: stringValue(config, "route_prefix"),
		AppName:     stringValue(config, "app_name"),
		TargetApp:   stringValue(config, "target_app"),
		AuthModule:  stringValue(config, "auth_module"),
		AuthzModule: stringValue(config, "authz_module"),
		Mode:        stringValue(config, "mode"),
		AssetRoot:   stringValue(config, "asset_root"),
	}
}
