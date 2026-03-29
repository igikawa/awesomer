# Developer Guide

Russian version: [developer.ru.md](./developer.ru.md)

This document explains how `awesomer` is structured at the code level and how data moves through the application. It is intended for contributors who need to change behaviour, extend features, or debug interactions between the TUI, daemon, and Linux-specific backends.

## Design Goals

The codebase is organized around a few practical goals:

- keep Linux process inspection isolated from UI code;
- keep resource-limiting backend details isolated from daemon and service logic;
- keep the TUI thin and event-driven;
- reuse process snapshots between the TUI and the daemon to reduce duplicate `/proc` scans;
- make side effects replaceable in tests through function variables and small interfaces.

## High-Level Architecture

At runtime the project is split into two entry points plus shared packages:

1. `cmd/awesomerctl`
   `awesomerctl` client entry point, config loading, logger setup, collector creation, and TUI startup.
2. `cmd/awesomerd`
   `awesomerd` entry point, config loading, daemon startup, IPC socket startup, and signal-driven shutdown.
3. `internal/collector`
   Shared short-lived cache for process snapshots and process trees.
4. `internal/service` and `internal/service/tui`
   User-facing process list, sorting, local process actions, and Bubble Tea event handling.
5. `internal/daemon`
   Background resource-control loop plus Unix-socket IPC that exposes daemon control to the client.

Support packages:

- `internal/config`
  YAML config model and defaults.
- `internal/daemon/info`
  Jail-state abstraction used both in-process and through the IPC client.
- `pkg/parser`
  Raw process inspection through `gopsutil`.
- `pkg/mutation`
  Low-level process mutation helpers such as CPU affinity and rlimit changes.
- `pkg/cgroups`
  Backend selection between `systemd` and direct `cgroup v2`.
- `pkg/logger`
  Main and daemon logger construction.

## Package Responsibilities

### `cmd/awesomerctl`

`main.go` is intentionally small orchestration code:

- ensure `config.yaml` exists;
- ensure a config file exists in the Linux config search path;
- load config;
- build the shared process collector;
- connect to the daemon socket if it exists;
- create loggers;
- run the TUI.

The startup path is written around replaceable function variables such as `loadConfigFn`, `newRemoteStateFn`, and `runTUIFn`. This keeps the client entry point testable without launching the real Bubble Tea program or a real daemon.

### `cmd/awesomerd`

`main.go` is the dedicated daemon launcher:

- ensure the root config path exists;
- refuse to run without root privileges;
- load config and require `daemon.run=true`;
- create `info.API`, collector, daemon logger, and daemon instance;
- expose daemon control over `/run/awesomer.sock`;
- run until `SIGINT` or `SIGTERM`.

### `internal/config`

This package owns the application config schema and defaults.

Important points:

- `ReadConfig()` reads YAML from a resolved config path and overlays it on top of defaults.
- `ResolveConfigPath()` is privilege-aware: non-root uses `~/.config/awesomer/config.yaml`, root uses `/etc/awesomer/config.yaml`.
- `DefaultConfig()` is the single place that defines default runtime values.
- the daemon sub-config and UI sub-config are nested, not flattened.

When adding a new option, update:

- the struct tags in config types;
- the default constructor;
- `internal/config/config.yaml.example`;
- user-facing docs.

### `internal/collector`

The collector is the shared source of process snapshots for the daemon and the TUI service layer.

It provides:

- `Processes()`
  Returns a short-lived cached slice of `parser.Info`.
- `ProcessTree(pid)`
  Builds and caches a `PPID -> children` map and post-order traversal for a root PID.

Important behaviour:

- cache lifetime is intentionally short;
- returned values are cloned before they leave the collector;
- storing a fresh process snapshot invalidates cached trees.

This package is the main place to optimize global process polling without leaking cache logic into UI or daemon code.

### `pkg/parser`

The parser is responsible for expensive raw process inspection.

It has two classes of APIs:

- cheap or relatively cheap snapshot APIs:
  - `AllProcesses()`
  - `ProcessTree()`
  - `ProcessInfo()`
- expensive detail API:
  - `HardObjectParse()`

`AllProcesses()` fans out `ProcessInfo()` calls with a bounded worker pool. The parser does not know anything about TUI tables, daemon thresholds, or process jail state.

### `internal/service`

The service package transforms raw process snapshots into data the TUI can render or act on.

Main responsibilities:

- sorting process rows;
- formatting process list rows for the table;
- remembering whether the rendered table state changed since the previous refresh;
- process actions such as stop/resume/kill;
- manual process jail toggling;
- rendering a simple ASCII process tree for the details panel.

`GetProcesses()` is intentionally more than a plain fetch:

- it reads the current shared snapshot from the collector;
- sorts it according to the selected mode;
- derives daemon marker state (`*`);
- compares a normalized row state with the previous render state;
- returns `changed=false` when the visible table would be identical.

This lets the TUI avoid unnecessary `SetRows()` calls.

### `internal/service/tui`

This package contains the Bubble Tea model and view logic.

The model is intentionally narrow:

- it does not read `/proc` directly;
- it delegates process list data to `serviceAPI`;
- it delegates detailed per-PID reads to `parserAPI`;
- it only handles UI state, key events, layout, and view rendering.

The two-panel layout is implemented with:

- a `table.Model` on the left;
- a `viewport.Model` on the right;
- Lip Gloss styles for focus and borders.

Key design choices:

- `tick()` returns `nil` when auto-refresh is disabled;
- `dataMsg` carries both rows and a `changed` flag;
- the table is only updated when `changed=true`;
- details and heavy process data are only loaded on demand.

### `internal/daemon`

The daemon is a polling loop with configuration hot reload.

Its responsibilities are:

- periodically read the shared process snapshot;
- track how many consecutive times a PID exceeded limits;
- ignore protected processes from the whitelist;
- avoid re-jailing processes already in jail;
- move full process trees into `processJail` after the threshold is reached;
- release jailed processes on shutdown;
- reload daemon config from the resolved Linux config path without restarting the app.

The daemon does not know how cgroup/systemd backend detection works. That is delegated to `pkg/cgroups`.

### `pkg/cgroups`

This package hides backend differences between:

- transient `systemd` units;
- direct `cgroup v2` filesystem operations.

Public operations are intentionally small and backend-neutral:

- create group;
- delete group;
- add process to group;
- move process back to root group;
- set resource rows/properties.

This is the package to change if resource backend behaviour needs to evolve.

## Runtime Flow

### Startup

There are now two startup flows.

Client startup:

1. `awesomerctl` resolves the Linux config path and ensures a file exists there.
2. Config is loaded through `internal/config`.
3. It creates a local collector for process snapshots.
4. If `/run/awesomer.sock` is available, it builds an IPC-backed jail state client.
5. The TUI starts and uses local process inspection plus optional remote daemon control.

Daemon startup:

1. `awesomerd` resolves `/etc/awesomer/config.yaml` and ensures it exists.
2. It loads config and validates root-only execution.
3. It creates `info.API`, a shared collector, and the daemon logger.
4. It starts the IPC socket server on `/run/awesomer.sock`.
5. It runs the enforcement loop until it receives a shutdown signal.

### Process List Refresh

The normal refresh path is:

1. TUI tick fires.
2. TUI asks `service.GetProcesses()`.
3. Service asks `collector.Processes()`.
4. Collector either returns cached processes or asks `parser.AllProcesses()`.
5. Service sorts and normalizes rows.
6. Service returns `rows` plus `changed`.
7. TUI updates the table only when `changed=true`.

### Detailed Process View

The compact and expanded details view intentionally bypass the collector:

- `formatedInfo()` uses `Parser.ProcessInfo(pid)`;
- `formatedBigInfo()` uses `Parser.HardObjectParse(pid)`;
- tree rendering uses `Parser.ProcessTree(pid)` and `Service.GetTuiTree(...)`.

This separation keeps the shared collector focused on the recurring hot path and avoids mixing cheap snapshots with expensive one-off detail queries.

### Daemon Enforcement Loop

The daemon loop is:

1. read current daemon config snapshot;
2. stop if config now says `run=false`;
3. read the shared process snapshot from the collector;
4. skip low PIDs and whitelist names;
5. skip processes already in jail;
6. increment a violation counter when CPU or RAM exceeds limits;
7. after repeated violations, fetch the process tree and move the full tree into `processJail`;
8. remove counters for PIDs that are no longer active.

Shutdown path:

- move all known jailed PIDs back to the root group;
- delete the resource group or transient unit;
- return the first cleanup error if one happened.

## Shared State and Concurrency

There are three notable shared-state objects:

### `collector.Collector`

- protected by an internal mutex;
- shared between daemon and service;
- owns cached snapshots and cached trees.

### `daemon.info.API`

- protected by its own mutex;
- tracks which PIDs are currently in jail;
- shared between daemon and service so the table marker and jail operations stay consistent.

### `service.Service`

- protected by an internal `RWMutex`;
- tracks sort mode;
- tracks previous normalized row state and previous rendered rows.

The codebase does not try to provide a fully reactive event bus. It uses polling plus short-lived shared snapshots because that model is simpler and predictable for Linux process monitoring.

## Error Handling

The codebase follows a few consistent rules:

- startup errors are returned up to `runApp()` and then handled at the top;
- polling loops log recoverable errors and continue;
- user-triggered actions usually log the error and keep the TUI alive;
- cleanup paths try to complete as much work as possible before returning an error.

In practice this means:

- process inspection failures should not crash the UI;
- daemon cleanup should still try to release all jailed PIDs even if one move fails.

## Testing Strategy

The project mostly uses package-level unit tests with injected dependencies.

Common testing patterns:

- function variables for side-effecting calls;
- small interfaces for parser/service/program dependencies;
- stub implementations instead of full mocks;
- direct testing of Bubble Tea model update paths.

Current coverage is strongest in:

- config loading;
- daemon lifecycle and hot reload;
- service sorting and mutations;
- TUI update paths;
- cgroup/systemd backend selection.

When adding new behaviour, prefer:

1. extend an existing interface or function variable seam;
2. add a focused unit test in the affected package;
3. keep integration-style TUI tests limited to event flow, not visual snapshots.

## Common Change Scenarios

### Add a New Config Option

1. Update config structs and defaults.
2. Thread the option into the package that owns the behaviour.
3. Update example config.
4. Update user docs.
5. Add a config parsing test and at least one behaviour test.

### Add a New TUI Action

1. Extend `serviceAPI` if the action belongs to business logic.
2. Add key handling in `Update()`.
3. Render user feedback in the details panel.
4. Add a TUI test for the event path.

### Change Resource Limiting Behaviour

1. Decide whether it belongs in daemon policy or backend mechanics.
2. If it changes limit policy, update `internal/daemon`.
3. If it changes backend implementation, update `pkg/cgroups`.
4. Keep backend-neutral function names unless platform-specific behaviour is unavoidable.

### Improve Performance

Start with these packages, in this order:

1. `internal/collector`
2. `internal/service`
3. `pkg/parser`
4. `internal/service/tui`

That order matters because most recurring cost is in process polling and row preparation, not in the rendering code itself.

## Development Notes

- The project targets Linux-first behaviour. Cross-platform abstractions are intentionally limited.
- The code prefers explicit dependencies over hidden package globals, but test seams still use function variables where that keeps changes smaller.
- `gopsutil` is convenient but not free. If profiling shows it dominates refresh cost, the next major optimization target is replacing selected paths with direct `/proc` reads.
- The TUI is intentionally separated from daemon policy. Keep it that way: the UI should ask for actions, not implement daemon logic itself.
