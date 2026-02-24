# workflow-plugin-admin

External plugin for the [workflow engine](https://github.com/GoCodeAlone/workflow) that provides the admin dashboard UI and config-driven admin routes.

## What it does

- Injects admin modules (HTTP server, JWT auth, SQLite, static file server) into the host engine config via `ConfigProvider`
- Embeds the admin UI assets and extracts them on startup
- Does not provide custom module types — all admin modules are native host types

## Build

```sh
make build
```

## Install

```sh
make install DESTDIR=/path/to/workflow
```

The plugin binary and assets are installed to `DESTDIR/data/plugins/workflow-plugin-admin/`.

## UI assets

The `ui_dist/` directory contains the built admin UI. To rebuild from source:

```sh
make ui
```

## Plugin SDK

This plugin follows the [workflow external plugin SDK](https://github.com/GoCodeAlone/workflow/tree/main/plugin/external/sdk) pattern. It implements `PluginProvider` and `ConfigProvider`.
