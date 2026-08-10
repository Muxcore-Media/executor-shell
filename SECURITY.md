# Security — executor-shell

This module runs shell commands (`sh -c`) on behalf of mesh callers. Treat it as a high-risk sidecar.

## Allowlist

| `EXECUTOR_ALLOWED_COMMANDS` | Behavior |
|-----------------------------|----------|
| unset / empty | **Deny all** (safe default) |
| comma-separated prefixes | Only payloads that start with a listed prefix |
| `*` | Unrestricted — **dev/test only** |

**Never set `EXECUTOR_ALLOWED_COMMANDS=*` in production.** Prefer the narrowest prefixes that match required automations (e.g. `/usr/local/bin/mux-backup,systemctl --user`).

## Timeouts and cancel

`Execute` uses `exec.CommandContext`. Callers must pass a deadline/cancel on the gRPC context; when the context is done the process is killed. Do not rely on unbounded mesh calls.

## Network / secrets

- Do not allow prefixes that can exfiltrate secrets (`curl`, `wget`, `scp`, broad `bash`/`sh`) unless the host is fully trusted and isolated.
- Run the module under a least-privilege OS user with no write access to MuxCore data dirs beyond what the task needs.
