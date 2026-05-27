package internal

import (
	"context"
	"fmt"

	"github.com/GoCodeAlone/workflow-plugin-admin/internal/contracts"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
	"google.golang.org/protobuf/types/known/structpb"
)

type adminStep struct {
	method string
	module string
}

func newAdminStep(method string, config map[string]any) *adminStep {
	module := "admin"
	if configured := stringValue(config, "module"); configured != "" {
		module = configured
	}
	return &adminStep{method: method, module: module}
}

func (s *adminStep) Execute(_ context.Context, _ map[string]any, _ map[string]map[string]any, current map[string]any, _ map[string]any, config map[string]any) (*sdk.StepResult, error) {
	args := mergeMaps(config, current)
	if args["module"] == nil {
		args["module"] = s.module
	}
	moduleName := stringValue(args, "module")
	module, err := lookupDashboardModule(moduleName)
	if err != nil {
		return nil, err
	}
	output, err := module.InvokeMethod(s.method, args)
	if err != nil {
		return nil, err
	}
	return &sdk.StepResult{Output: output}, nil
}

func mergeMaps(maps ...map[string]any) map[string]any {
	out := make(map[string]any)
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

func typedRegisterContribution(ctx context.Context, req sdk.TypedStepRequest[*contracts.AdminStepConfig, *contracts.RegisterContributionInput]) (*sdk.TypedStepResult[*contracts.RegisterContributionOutput], error) {
	moduleName := req.Config.GetModule()
	if req.Input.GetModule() != "" {
		moduleName = req.Input.GetModule()
	}
	module, err := lookupDashboardModule(moduleName)
	if err != nil {
		return nil, err
	}
	if err := module.registry.register(req.Input.GetContribution()); err != nil {
		return &sdk.TypedStepResult[*contracts.RegisterContributionOutput]{Output: &contracts.RegisterContributionOutput{Error: err.Error()}}, nil
	}
	return &sdk.TypedStepResult[*contracts.RegisterContributionOutput]{
		Output: &contracts.RegisterContributionOutput{Registered: true, Contribution: req.Input.GetContribution()},
	}, nil
}

func typedListContributions(_ context.Context, req sdk.TypedStepRequest[*contracts.AdminStepConfig, *contracts.ListContributionsInput]) (*sdk.TypedStepResult[*contracts.ListContributionsOutput], error) {
	moduleName := req.Config.GetModule()
	if req.Input.GetModule() != "" {
		moduleName = req.Input.GetModule()
	}
	module, err := lookupDashboardModule(moduleName)
	if err != nil {
		return nil, err
	}
	return &sdk.TypedStepResult[*contracts.ListContributionsOutput]{
		Output: &contracts.ListContributionsOutput{Contributions: module.registry.listForPermissions(req.Input.GetAppContext(), req.Input.GetGrantedPermissions())},
	}, nil
}

func typedAuthorizeAction(_ context.Context, req sdk.TypedStepRequest[*contracts.AdminStepConfig, *contracts.AuthorizeAdminActionInput]) (*sdk.TypedStepResult[*contracts.AuthorizeAdminActionOutput], error) {
	allowed := req.Input.GetAuthzChecked() && req.Input.GetAuthzAllowed()
	reason := "authz allowed"
	if !req.Input.GetAuthzChecked() {
		reason = "missing authz evidence"
	} else if !req.Input.GetAuthzAllowed() {
		reason = "authz denied"
	}
	return &sdk.TypedStepResult[*contracts.AuthorizeAdminActionOutput]{
		Output: &contracts.AuthorizeAdminActionOutput{
			Allowed:  allowed,
			Subject:  req.Input.GetSubject(),
			Resource: req.Input.GetResource(),
			Action:   req.Input.GetAction(),
			Reason:   reason,
		},
	}, nil
}

func typedResourceAction(_ context.Context, req sdk.TypedStepRequest[*contracts.AdminStepConfig, *contracts.AdminResourceActionInput]) (*sdk.TypedStepResult[*contracts.AdminResourceActionOutput], error) {
	if !req.Input.GetAuthzChecked() || !req.Input.GetAuthzAllowed() {
		reason := "authz denied"
		if !req.Input.GetAuthzChecked() {
			reason = "missing authz evidence"
		}
		return &sdk.TypedStepResult[*contracts.AdminResourceActionOutput]{
			Output: &contracts.AdminResourceActionOutput{
				Result: &contracts.AdminActionResult{Accepted: false, Status: "denied", Error: reason},
			},
		}, nil
	}
	return &sdk.TypedStepResult[*contracts.AdminResourceActionOutput]{
		Output: &contracts.AdminResourceActionOutput{
			Result: &contracts.AdminActionResult{
				Accepted: true,
				Status:   "accepted",
				Output:   emptyStruct(),
			},
		},
	}, nil
}

func emptyStruct() *structpb.Struct {
	s, err := structpb.NewStruct(map[string]any{})
	if err != nil {
		panic(fmt.Sprintf("build empty struct: %v", err))
	}
	return s
}
