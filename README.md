# Awesomer

Russian version: [ru_README.md](./ru_README.md)
Developer guide: [docs/developer.md](./docs/developer.md)

`Awesomer` is a Linux terminal utility for process monitoring and control. It shows a live process list, lets you inspect detailed information for the selected process, and can run a background daemon that automatically moves heavy processes into a constrained resource group. If the target system uses `systemd`, limits are managed through a transient unit; otherwise the application falls back to direct `cgroup v2` usage.

## Contents

1. [Features](#features)
2. [Stack](#stack)
3. [System Requirements](#system-requirements)
4. [Project Structure](#project-structure)
5. [Installation and Run](#installation-and-run)
6. [Configuration](#configuration)
7. [How the Interface Works](#how-the-interface-works)
8. [How the Daemon Works](#how-the-daemon-works)
9. [Logs](#logs)
10. [Limitations](#limitations)

## Features

- Live process list refresh with configurable update frequency.
- Sorting by PID, name, CPU, memory, thread count, and user.
- Compact process card with command line, owner, and `nice`.
- Extended details:
  - network connections,
  - open files,
  - child process tree.
- Direct process control from the TUI:
  - stop,
  - resume,
  - kill,
  - kill full process tree,
  - change CPU affinity,
  - change `RLIMIT_NOFILE`,
  - manually move a process into `processJail` or return it back.
- Background daemon for resource control and automatic jailing through a `systemd` unit or `cgroup v2`.
- Separate logs for the main application and the daemon.

## Stack

- Go `1.25`
- Bubble Tea for the terminal UI
- Bubbles (`table`, `viewport`) for widgets
- Lip Gloss for layout and styling
- `gopsutil` for process inspection
- `yaml.v3` for loading `config.yaml`
- `systemd` units or Linux `cgroup v2` for resource limiting

## System Requirements

The project targets Linux.

Minimum requirements:

- Linux with accessible `/proc`
- Linux with `cgroup v2` mounted at `/sys/fs/cgroup`
- `systemd` if limits should be created through units
- Go `1.25` or newer

Practical considerations:

- Sending signals to processes owned by other users requires sufficient permissions.
- Creating and configuring resource limits through `systemd` or `cgroup v2` requires elevated privileges.
- Without the required permissions, process inspection may still work while process control and daemon resource limiting may fail.

## Project Structure

```text
cmd/project/main.go          Entry point
internal/config              Shared application configuration
internal/daemon              Monitoring daemon and hot-reload logic
internal/daemon/config       Daemon configuration model
internal/daemon/info         Shared API for "process in jail" state
internal/service             TUI business logic, sorting, and actions
internal/service/tui         Bubble Tea interface
pkg/parser                   Process parsing through gopsutil
pkg/cgroups                  Resource backend selection: systemd unit or cgroup v2
pkg/logger                   Application and daemon loggers
pkg/mutation                 Low-level process mutation helpers
```

## Installation and Run

### Build

```bash
git clone https://github.com/igikawa/awesomer.git
cd awesomer
go build -o awesomer ./cmd/project
```

### Quick Start

```bash
cp internal/config/config.yaml.example config.yaml
./awesomer
```

If you plan to use the daemon and resource limiting, run the application as a user that has access to `systemd` and/or `/sys/fs/cgroup`, and to the target processes.

### Run Without Building First

```bash
cp internal/config/config.yaml.example config.yaml
go run ./cmd/project
```

### What Happens on Startup

- The application reads `config.yaml` from the repository root.
- If `config.yaml` does not exist, the program creates an empty file and uses default values.
- If `daemon.run: true`, the background daemon starts together with the TUI.

## Configuration

Example `config.yaml`:

```yaml
tick: 1

logger:
  log_path: ./awesome.log
  daemon_log_path: ./awesome.daemon.log

daemon:
  run: true
  tick: 5
  cpu_limit: 85
  ram_limit: 60
  cpu_quota: 20
  ram_quota: 8G
  whitelist:
    - systemd
    - sshd

ui:
  table_width: 0
  info_width: 72
  border_color: "102"
  active_border_color: "62"
  selection_text_color: "229"
  selection_background_color: "57"
```

### General Parameters

- `tick`: process list refresh frequency in seconds.
  - `0` disables auto-refresh.
  - Default: `1`

### Logging

- `logger.log_path`: path to the main application log.
  - Default: `./awesome.log`
- `logger.daemon_log_path`: path to the daemon log.
  - Default: `./awesome.daemon.log`

### Daemon Parameters

- `daemon.run`: whether to start the daemon together with the TUI.
  - Default: `false`
- `daemon.tick`: daemon polling interval in seconds.
  - Default: `3`
- `daemon.cpu_limit`: CPU threshold after which a process receives a warning.
  - Default: `85`
- `daemon.ram_limit`: RAM usage threshold in percent after which a process receives a warning.
  - Default: `60`
- `daemon.cpu_quota`: CPU limit for the constrained resource group.
  - Mapped to `CPUQuota` when `systemd` is used.
  - Written to `cpu.max` for direct `cgroup v2`.
  - Default: `20`
- `daemon.ram_quota`: memory limit for the constrained resource group.
  - Mapped to `MemoryMax` when `systemd` is used.
  - Written to `memory.max` for direct `cgroup v2`.
  - Default: `8G`
- `daemon.whitelist`: process names that the daemon must never move into the constrained group.
  - Matching is case-insensitive and based on the process name shown in the table.
  - Intended for critical services that an administrator wants to protect from automatic limiting.
  - Default: `["systemd", "sshd"]`

The daemon hot-reloads `config.yaml`, so configuration changes are applied without restarting the application.

### UI Parameters

- `ui.table_width`: preferred table panel width in characters.
  - `0` means width is derived automatically from the total terminal width.
  - Default: `0`
- `ui.info_width`: preferred details panel width in characters.
  - Used when `ui.table_width` is not set explicitly.
  - Default: `72`
- `ui.border_color`: border color for inactive panels.
  - Default: `"102"`
- `ui.active_border_color`: border color for the focused panel.
  - Default: `"62"`
- `ui.selection_text_color`: text color for the selected table row.
  - Default: `"229"`
- `ui.selection_background_color`: background color for the selected table row.
  - Default: `"57"`

## How the Interface Works

The interface consists of two panels:

- left: process table;
- right: details panel.

At startup, the right panel shows built-in help. The process list refreshes by the `tick` timer when `tick != 0`.

### Table Columns

- `PID`
- `Name`
- `CPU`
- `Mem`
- `Threads`
- `User`
- `*`: the process is controlled by the daemon and already placed into the constrained group

### Navigation

- `↑` / `↓`: move through the process list
- `Tab`: switch focus between the table and the details panel
- `Esc`: return focus to the table
- `q` or `Ctrl+C`: graceful shutdown

### Sorting

- `p`: by PID
- `n`: by name
- `c`: by CPU
- `m`: by memory
- `t`: by thread count
- `u`: by user

### Process Actions

- `Enter`: show compact information for the selected process
- `h`: show extended information
- `a`: set CPU affinity for the selected process
- `l`: change `RLIMIT_NOFILE` for the selected process
- `j`: manually move the selected process and its tree into `processJail` or return it back
- `s`: send `SIGSTOP`
- `r`: send `SIGCONT`
- `k`: send `SIGKILL`
- `d`: kill the full process tree rooted at the selected PID

### What Is Shown on the Right

`Enter` mode:

- PID
- process name
- user
- `nice`
- CPU_AFFINITY
- RLIMIT_NOFILE
- full command line

`h` mode:

- network connections
- open files
- child process tree

`a` mode:

- CPU core list input in the `0,1,3` format
- applied through `SchedSetaffinity`

`l` mode:

- new `RLIMIT_NOFILE` value input
- applied through `prlimit`

## How the Daemon Works

If `daemon.run: true`, the application starts the background daemon together with the TUI.

Algorithm:

1. The daemon selects the resource backend.
2. If `systemd` is active, it creates a transient `processJail.service` unit; otherwise it creates a direct `processJail` cgroup.
3. CPU and memory limits are applied to the group.
4. The daemon periodically scans all processes.
5. Some processes are skipped: PID `< 100`, `systemd`, and `sshd`.
6. If a process exceeds `daemon.cpu_limit` or `daemon.ram_limit`, it receives a warning.
7. After 3 warnings, the process and its child processes are moved into `processJail`.
8. Such processes are marked with `*` in the table.

In addition to automatic mode, a process can be sent to `processJail` manually with the `j` key. Manual moves use the same `daemon.cpu_quota` and `daemon.ram_quota` values.

When the application shuts down or `daemon.run` is switched to `false`, the daemon first returns jailed processes to the root group and then removes the created `systemd` unit or `cgroup`.

## Logs

By default the project writes to two files:

- `./awesome.log`: main application and TUI events
- `./awesome.daemon.log`: daemon events

Both loggers use the standard Go `log.Logger` with date, time, and short file position.

## Limitations

- The project only works on Linux.
- The code depends on `/proc`, POSIX signals, and Linux resource limiting mechanisms (`systemd` or `cgroup v2`).
- Extended information for some processes (`connections`, `open files`) may be unavailable because of permissions.
- The daemon needs permission to work with `systemd` or write access to `/sys/fs/cgroup`.
