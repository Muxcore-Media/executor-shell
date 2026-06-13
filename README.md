# Executor Shell

[![Go Version](https://img.shields.io/badge/Go-1.26-blue)](https://go.dev/)
[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)

**Generic shell command executor for the MuxCore worker pool.**

A MuxCore sidecar module that executes shell commands from worker pool tasks. When a task arrives with type `"shell"`, it runs the payload as a shell command and returns the output.

---

## How It Works

```
Worker pool task ──→ executor-shell (gRPC) ──→ sh -c "<payload>"
                                                   │
                                                   ▼
                                             stdout + stderr
```

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `EXECUTOR_GRPC_ADDR` | `:9602` | gRPC listen address |
| `EXECUTOR_ALLOWED_COMMANDS` | `""` | Comma-separated command prefixes (empty = allow any) |

### Security

When `EXECUTOR_ALLOWED_COMMANDS` is set (e.g., `git,make,ls`), only commands starting with one of the specified prefixes are executed. Rejected commands are logged at WARN level.

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

### Systemd

Copy the service unit and binary:

```bash
cp deploy/systemd/muxcore-module.service /etc/systemd/system/executor-shell.service
cp executor-shell /usr/local/bin/executor-shell
systemctl daemon-reload
systemctl enable --now executor-shell
```
