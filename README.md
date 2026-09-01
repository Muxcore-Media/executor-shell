# Executor Shell

[![Go Version](https://img.shields.io/badge/Go-1.26-blue)](https://go.dev/)
[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)

**Generic shell command executor for the MuxCore worker pool.**

A MuxCore sidecar module that executes shell commands from worker pool tasks. When a task arrives with type `"shell"`, core's worker-pool dispatcher calls this module's mesh method `Execute` with a JSON `contracts.WorkerTask`; the module runs `task.Payload` as a argv-split command (no `sh -c`).

---

## How It Works

```
Worker pool task ──→ core mesh Call("Execute", WorkerTask JSON)
                           │
                           ▼
              executor-shell ModuleMesh (:9605)
                           │
                           ▼
              exec(first_token, remaining_argv...)
                           │
                           ▼
              success → Result = stdout (bounded)
              failure → Error = stderr (or exec error)
```

**Capability:** `executor.shell`  
**Contract:** `Executor` (`v0.4.0`, min core `0.4.0`)  
**Task type:** `shell`  
**MVP stack:** optional — set `MVP_ENABLE_EXECUTOR_SHELL=1` in `_mvp/.env` or export before `./run-host.sh up` (see `_mvp/run-host.sh`).

Admin UI settings dial this module's **`Info().HTTPAddr`** gRPC listen address (same port as `EXECUTOR_GRPC_ADDR`, default `:9605`) for `Settings` / `UpdateSetting` mesh calls. That address is **not** core's mesh; it is this sidecar's ModuleMesh + ExecutorService endpoint.

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `EXECUTOR_GRPC_ADDR` | `:9605` | gRPC listen address (`Info().HTTPAddr` / admin-ui dial target) |
| `EXECUTOR_ALLOWED_COMMANDS` | `""` (deny all) | Comma-separated **first tokens** (command names). Empty denies every command. |
| `EXECUTOR_ALLOW_WILDCARD` | unset | Set to `1` / `true` to allow `*` in env or live `allowed_commands` settings. Without it, `*` is rejected. |
| `EXECUTOR_TIMEOUT` | `5m` | Default max runtime when the mesh context has no deadline (worker-pool dispatcher uses a long-lived ctx). Also updatable via settings key `timeout`. |
| `EXECUTOR_MAX_OUTPUT` | `1048576` (1 MiB) | Max combined stdout/stderr capture per command; overflow fails closed. |
| `MUXCORE_GRPC_ADDR` | — | Core mesh address (required). |
| `MUXCORE_MODULE_ID` | `executor-shell` | Module ID override. |
| `MUXCORE_INSECURE_DISABLE_TLS` | — | Dev-only plaintext to core. |

### Live settings (admin-ui)

| Key | Description |
|-----|-------------|
| `allowed_commands` | Same as `EXECUTOR_ALLOWED_COMMANDS`; applied immediately. `*` requires `EXECUTOR_ALLOW_WILDCARD=1`. |
| `timeout` | Same as `EXECUTOR_TIMEOUT` (Go duration, e.g. `5m`, `30s`). |

### Security

Commands are checked before execution:

- **Empty allowlist** — deny all (fail-closed default).
- **First-token match** — e.g. `git,make` allows only commands whose argv[0] is `git` or `make` (not prefix/substring match).
- **No shell metacharacters** — `;|&$()`\`<>` and newlines are rejected so `echo` cannot chain into `rm -rf /`.
- **`*`** — unrestricted first-token check only when `EXECUTOR_ALLOW_WILDCARD=1` is set (dev/test; never in production).

Rejected commands are logged at WARN with a truncated SHA-256 fingerprint, not the full payload.

See [SECURITY.md](SECURITY.md).

---

## Development

```bash
make build   # compile the module binary
make test    # run tests with race detection
make lint    # golangci-lint
make fmt     # format Go source
make tidy    # go mod tidy
```

## Deployment

### Docker

```bash
make docker      # build image
make docker-push # build and push
```

Compose publishes gRPC on **9605** and sets `EXECUTOR_ALLOW_WILDCARD=1` when using `EXECUTOR_ALLOWED_COMMANDS=*`.

### Systemd

Copy the service unit and binary:

```bash
cp deploy/systemd/muxcore-module.service /etc/systemd/system/executor-shell.service
cp executor-shell /usr/local/bin/executor-shell
systemctl daemon-reload
systemctl enable --now executor-shell
```
