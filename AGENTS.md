# AGENTS.md — executor-shell

MuxCore sidecar module (`executor-shell`). Workspace deploy and SSH: [`../AGENTS.md`](../AGENTS.md). Default ports: [`_mvp/PORTS.md`](../_mvp/PORTS.md).

## Module identity

| Field | Value |
|-------|-------|
| Directory | `executor-shell` |
| Capabilities | `executor.shell`, `settings` (see muxcore.json) |
| Contracts | `core/pkg/contracts` **Executor** v0.4.0 |

Optional on the MVP stack: `MVP_ENABLE_EXECUTOR_SHELL=1` in `_mvp/.env` (see `_mvp/run-host.sh`).

## Agent rules

- Modules run as gRPC sidecars; capabilities are the security boundary.
- TLS required in production (`MUXCORE_INSECURE_DISABLE_TLS` is dev-only).
- Match existing Go patterns; run `gofmt` and package tests before finishing.
- Cross-module events: prefer `github.com/Muxcore-Media/contracts-media/events` over deprecated `core/pkg/contracts` aliases.
- Do not edit polluted workspace dumps (see `MASTER-ROADMAP.md` Appendix H).

## Build

```bash
cd executor-shell
nix-shell -p go --run 'go test ./...'
```
