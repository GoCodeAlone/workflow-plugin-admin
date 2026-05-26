# workflow-plugin-admin

External Workflow plugin that provides an embedded admin mini-app, a typed
`admin.dashboard` module, and strict protobuf contracts for contributed admin
surfaces.

## What It Does

- Hosts a runtime admin contribution registry through `admin.dashboard`.
- Lets workflows/plugins register admin surfaces with
  `step.admin_register_contribution`.
- Lists contributed surfaces with `step.admin_list_contributions`.
- Enforces default-deny admin action behavior with
  `step.admin_authorize_action` and upstream authz evidence.
- Provides a static admin shell that can render built-in identity,
  authorization, and contribution panels.
- Advertises module, step, and service-method contracts through
  `plugin.contracts.json` and `ContractRegistry()`.

Authentication and authorization are intentionally composed with other Workflow
plugins. Use `auth.jwt` or another auth plugin for identity, then run
`workflow-plugin-authz` steps before admin action/list steps. Admin consumes the
authorization evidence and denies missing evidence by default.

## Contracts

Module:

- `admin.dashboard`

Steps:

- `step.admin_register_contribution`
- `step.admin_list_contributions`
- `step.admin_authorize_action`
- `step.admin_resource_action`

Service methods on `admin.dashboard`:

- `RegisterContribution`
- `ListContributions`
- `AuthorizeAction`
- `DispatchResourceAction`

The protobuf package is `workflow.plugins.admin.v1`.

## Minimal Composition

```yaml
modules:
  - name: auth
    type: auth.jwt
    config:
      issuer: my-app

  - name: authz
    type: authz.casbin

  - name: admin
    type: admin.dashboard
    config:
      route_prefix: /admin
      app_name: My App
      target_app: my-app
      auth_module: auth
      authz_module: authz

pipelines:
  register-admin-surface:
    trigger:
      type: manual
    steps:
      - name: register
        type: step.admin_register_contribution
        config:
          module: admin
        input:
          id: orders
          title: Orders
          path: /admin/orders
          render_mode: json-schema
```

See `examples/minimal/config.yaml` and workflow-scenarios scenario
`89-admin-dashboard` for the app + admin composition pattern.

## Build

```sh
make build
```

## Test

```sh
GOWORK=off go test ./...
```

## Install

```sh
make install DESTDIR=/path/to/workflow
```

The install target copies the plugin binary, `plugin.json`,
`plugin.contracts.json`, and embedded UI assets to
`DESTDIR/data/plugins/workflow-plugin-admin/`.

## UI Assets

`internal/ui_dist/index.html` contains the embedded admin shell. It fetches
contributions from `/api/admin/contributions` by default; compose that endpoint
with Workflow routes/pipelines backed by admin steps and auth/authz checks.

