package internal

import (
	"testing"

	"github.com/GoCodeAlone/workflow/wftest"
)

// TestAdminPlugin_BasicPipeline verifies that a pipeline can be built and
// executed with the wftest harness in this plugin's context.
func TestAdminPlugin_BasicPipeline(t *testing.T) {
	rec := wftest.RecordStep("step.admin_process")
	h := wftest.New(t, wftest.WithYAML(`
pipelines:
  admin-task:
    trigger:
      type: manual
    steps:
      - name: process
        type: step.admin_process
        config:
          action: list
`), rec)

	result := h.ExecutePipeline("admin-task", map[string]any{"user": "admin"})
	if result.Error != nil {
		t.Fatalf("pipeline failed: %v", result.Error)
	}
	if rec.CallCount() != 1 {
		t.Errorf("expected 1 call, got %d", rec.CallCount())
	}
}

// TestAdminPlugin_PipelineWithOutput verifies that step output flows through
// the pipeline correctly.
func TestAdminPlugin_PipelineWithOutput(t *testing.T) {
	rec := wftest.RecordStep("step.admin_fetch").WithOutput(map[string]any{
		"status": "ok",
		"count":  42,
	})

	h := wftest.New(t, wftest.WithYAML(`
pipelines:
  admin-fetch:
    trigger:
      type: manual
    steps:
      - name: fetch
        type: step.admin_fetch
        config:
          resource: users
`), rec)

	result := h.ExecutePipeline("admin-fetch", map[string]any{"request_id": "test-123"})
	if result.Error != nil {
		t.Fatalf("pipeline failed: %v", result.Error)
	}
	if rec.CallCount() != 1 {
		t.Errorf("expected 1 call, got %d", rec.CallCount())
	}
	calls := rec.Calls()
	if calls[0].Input["request_id"] != "test-123" {
		t.Errorf("expected input request_id=test-123, got %v", calls[0].Input["request_id"])
	}
	if calls[0].Config["resource"] != "users" {
		t.Errorf("expected config resource=users, got %v", calls[0].Config["resource"])
	}
}

// TestAdminPlugin_MultiStepPipeline verifies that multiple steps execute in
// sequence and each receives the accumulated pipeline state.
func TestAdminPlugin_MultiStepPipeline(t *testing.T) {
	recValidate := wftest.RecordStep("step.admin_validate").WithOutput(map[string]any{"validated": true})
	recProcess := wftest.RecordStep("step.admin_execute").WithOutput(map[string]any{"executed": true})

	h := wftest.New(t, wftest.WithYAML(`
pipelines:
  admin-workflow:
    trigger:
      type: manual
    steps:
      - name: validate
        type: step.admin_validate
        config:
          strict: true
      - name: execute
        type: step.admin_execute
        config:
          mode: async
`), recValidate, recProcess)

	result := h.ExecutePipeline("admin-workflow", map[string]any{"payload": "test"})
	if result.Error != nil {
		t.Fatalf("pipeline failed: %v", result.Error)
	}
	if recValidate.CallCount() != 1 {
		t.Errorf("expected validate step called once, got %d", recValidate.CallCount())
	}
	if recProcess.CallCount() != 1 {
		t.Errorf("expected execute step called once, got %d", recProcess.CallCount())
	}

	// The second step should see the output from the first step merged into its input.
	executeCalls := recProcess.Calls()
	if executeCalls[0].Input["validated"] != true {
		t.Errorf("expected execute step to see validated=true in input, got %v", executeCalls[0].Input["validated"])
	}
}
