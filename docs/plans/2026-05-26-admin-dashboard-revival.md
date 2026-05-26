# Admin Dashboard Revival Implementation Plan

> **For the implementing agent:** REQUIRED SUB-SKILL: Use autodev:executing-plans to implement this plan task-by-task.

**Goal:** Build a contract-first Workflow admin plugin that hosts contributed admin surfaces, composes auth/authz through Workflow, and proves the slice with workflow-scenarios.

**Architecture:** Add strict protobuf contracts and an `admin.dashboard` remote module with a runtime contribution registry. Expose typed steps/service methods for registration, listing, authorization evidence checks, and resource-action envelopes; update the embedded shell and scenario fixtures to exercise the real plugin boundary.

**Tech Stack:** Go 1.26, Workflow external plugin SDK, protobuf/protoc-gen-go, embedded HTML/CSS/JS, bash scenario tests.

**Base branch:** main

---

## Scope Manifest

**PR Count:** 1
**Tasks:** 6
**Estimated Lines of Change:** ~1700

**Out of scope:**
- Replacing `workflow-plugin-auth` or `workflow-plugin-authz`; admin composes their current primitives.
- Adding new cloud resources or persistent admin-owned storage.

**PR Grouping:**

| PR # | Title | Tasks | Branch |
|------|-------|-------|--------|
| 1 | Admin dashboard revival foundation | Task 1, Task 2, Task 3, Task 4, Task 5, Task 6 | feat/admin-dashboard-revival |

**Status:** Complete 2026-05-26T23:21:38Z

## Design Trace

| design requirement | plan tasks |
|---|---|
| Strict proto contracts | Task 1, Task 3 |
| `admin.dashboard` module + registry | Task 2 |
| Auth/authz composition + default deny | Task 2, Task 3, Task 5 |
| Usable admin mini-app shell | Task 4 |
| Plugin contribution pattern | Task 2, Task 3, Task 6 |
| Scenario proof | Task 5 |
| Rollback and docs | Task 6 |

### Task 1: Strict Admin Proto Contracts

**Files:**
- Create: `internal/contracts/admin.proto`
- Generate: `internal/contracts/admin.pb.go`
- Modify: `go.mod`, `go.sum` only if generation requires dependency metadata updates.
- Test: `internal/plugin_contracts_test.go`

**Step 1: Write failing contract tests**

Add tests asserting `NewAdminPlugin()` implements `ContractRegistry()`,
declares module `admin.dashboard`, declares steps:
`step.admin_register_contribution`, `step.admin_list_contributions`,
`step.admin_authorize_action`, `step.admin_resource_action`, and includes
`workflow.plugins.admin.v1.AdminDashboardConfig`.

Run: `go test ./internal -run TestAdminPlugin_ContractRegistry -count=1`
Expected: FAIL because `ContractRegistry` and proto types are missing.

**Step 2: Add proto schema**

Create `admin.proto` with messages:
`AdminDashboardConfig`, `AdminContribution`, `AdminPermission`,
`RegisterContributionInput/Output`, `ListContributionsInput/Output`,
`AuthorizeAdminActionInput/Output`, `AdminResourceActionInput/Output`,
and `AdminActionResult`.

**Step 3: Generate Go**

Run: `PATH="$PATH:/Users/jon/go/bin" protoc --go_out=. --go_opt=paths=source_relative internal/contracts/admin.proto`
Expected: `internal/contracts/admin.pb.go` created.

**Step 4: Verify**

Run: `go test ./internal -run TestAdminPlugin_ContractRegistry -count=1`
Expected: PASS.

Rollback: remove generated contract files and contract descriptors; `go test ./internal` should return to previous behavior.

### Task 2: `admin.dashboard` Module Registry

**Files:**
- Create: `internal/module_dashboard.go`
- Create: `internal/registry.go`
- Modify: `internal/plugin.go`
- Test: `internal/module_dashboard_test.go`

**Step 1: Write failing registry tests**

Test that:
- invalid contribution IDs/paths are rejected,
- duplicate contributions update in place by ID,
- list returns deterministic order,
- `InvokeMethod("RegisterContribution")` and `InvokeMethod("ListContributions")` work through `sdk.ServiceInvoker`,
- missing authz evidence denies `AuthorizeAction`.

Run: `go test ./internal -run 'TestDashboard|TestContributionRegistry' -count=1`
Expected: FAIL because module/registry do not exist.

**Step 2: Implement registry and module**

Add `admin.dashboard` as legacy and typed module type. Store dashboard modules
in a concurrency-safe registry keyed by module name. Implement service methods:
`RegisterContribution`, `ListContributions`, `AuthorizeAction`,
`DispatchResourceAction`.

**Step 3: Verify**

Run: `go test ./internal -run 'TestDashboard|TestContributionRegistry' -count=1`
Expected: PASS.

Rollback: remove `admin.dashboard` module registration; config-provider-only mode remains.

### Task 3: Admin Typed Steps and Contract Parity

**Files:**
- Create: `internal/step_admin.go`
- Modify: `internal/plugin.go`
- Create: `plugin.contracts.json`
- Modify: `plugin.json`
- Test: `internal/step_admin_test.go`, `internal/plugin_contracts_test.go`

**Step 1: Write failing step tests**

Test register/list/authorize/resource-action steps through legacy execution and
typed providers. Assert `plugin.contracts.json` exactly matches runtime
`ContractRegistry()` descriptors.

Run: `go test ./internal -run 'TestAdminSteps|TestPluginContractsJSON' -count=1`
Expected: FAIL because steps and descriptor file are missing.

**Step 2: Implement steps**

Add step types:
- `step.admin_register_contribution`
- `step.admin_list_contributions`
- `step.admin_authorize_action`
- `step.admin_resource_action`

`step.admin_authorize_action` consumes explicit upstream authz evidence fields
and defaults to denied when evidence is absent or false.

**Step 3: Advertise capabilities**

Update `plugin.json` capabilities and add `plugin.contracts.json`.

**Step 4: Verify**

Run: `go test ./internal -run 'TestAdminSteps|TestPluginContractsJSON' -count=1`
Expected: PASS.

Rollback: remove advertised module/step types and descriptor file; old plugin manifest remains loadable.

### Task 4: Embedded Admin Shell

**Files:**
- Modify: `internal/ui_dist/index.html`
- Modify: `README.md`
- Test: `internal/assets_test.go`

**Step 1: Write failing shell tests**

Test embedded `index.html` contains:
- `data-admin-shell-version`,
- contribution navigation root,
- auth/authz management sections,
- configurable `data-contributions-endpoint`,
- no hardcoded secret names.

Run: `go test ./internal -run TestEmbeddedAdminShell -count=1`
Expected: FAIL because current shell is placeholder/old asset.

**Step 2: Implement shell**

Replace the embedded HTML with a usable admin app shell: sidebar, overview,
contributions, identity, authorization, and action log panels. It must read
contributions from a configurable endpoint defaulting to
`/api/admin/contributions`; scenario Task 5 wires that endpoint through Workflow
routes/pipelines backed by admin steps.

**Step 3: Verify**

Run: `go test ./internal -run TestEmbeddedAdminShell -count=1`
Expected: PASS.

Rollback: revert `internal/ui_dist/index.html`; API contracts remain.

### Task 5: Workflow Scenario Proof

**Files:**
- Create: `/Users/jon/workspace/workflow-scenarios/scenarios/89-admin-dashboard/config/app.yaml`
- Create: `/Users/jon/workspace/workflow-scenarios/scenarios/89-admin-dashboard/scenario.yaml`
- Create: `/Users/jon/workspace/workflow-scenarios/scenarios/89-admin-dashboard/test/run.sh`
- Modify: `/Users/jon/workspace/workflow-scenarios/scenarios.json`

**Step 1: Write scenario files**

Add a scenario that has:
- primary app health endpoint,
- auth route using `auth.jwt`,
- authz module/policy,
- admin dashboard module,
- static admin shell mounted under `/admin`,
- pipeline registering a sample app contribution,
- `/api/admin/contributions` route protected by auth/authz evidence and backed
  by `step.admin_list_contributions`.

**Step 2: Verify static scenario contract**

Run: `bash scenarios/89-admin-dashboard/test/run.sh --static`
Expected: PASS lines for config, admin module, auth/authz use, contribution route, and deny test definitions.

**Step 3: Runtime proof when cluster is available**

Run: `make test SCENARIO=89-admin-dashboard`
Expected if minikube/workflow image available: admin endpoint returns registered contribution and unauthorized request is denied. If environment unavailable, record blocker in final evidence.

Rollback: remove scenario directory and `scenarios.json` entry; no runtime resource remains.

### Task 6: Full Verification and Documentation

**Files:**
- Modify: `README.md`
- Modify: `examples/minimal/config.yaml`
- Modify: `docs/plans/2026-05-26-admin-dashboard-revival-design.md` only for implementation backports if assumptions changed.

**Step 1: Update docs/example**

Document module/step contracts, auth/authz composition pattern, rendering
modes, and minimal YAML. Keep README concise.

**Step 2: Run plugin verification**

Run:
`go test ./... -count=1`
`go build ./cmd/workflow-plugin-admin`
Expected: both exit 0.

**Step 3: Strict contract audit when wfctl is available**

Run from `/Users/jon/workspace/workflow`:
`go run ./cmd/wfctl plugin validate-contract --file /Users/jon/workspace/workflow-plugin-admin/plugin.json --strict-contracts`
Expected: exit 0 or document exact command mismatch if wfctl CLI surface differs.

**Step 4: Runtime launch validation**

Run built plugin through the SDK smoke path if a local host loader is available;
otherwise use `go test ./internal -run TestAdminPlugin_Interfaces -count=1`
plus `go build` as plugin-boundary evidence and record the limitation.

Rollback: revert docs/example updates; no runtime state.
