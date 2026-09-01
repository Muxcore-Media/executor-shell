# Security — executor-shell

This module runs OS commands on behalf of mesh callers. Treat it as a high-risk sidecar.

## Allowlist

| `EXECUTOR_ALLOWED_COMMANDS` | Behavior |
|-----------------------------|----------|
| unset / empty | **Deny all** (safe default) |
| comma-separated first tokens | Only commands whose argv[0] exactly matches a listed token |
| `*` | Unrestricted token check — **only when `EXECUTOR_ALLOW_WILDCARD=1`** |

**Never set `EXECUTOR_ALLOWED_COMMANDS=*` in production** without understanding the blast radius. Prefer the narrowest command names that match required automations (e.g. `git,make,/usr/local/bin/mux-backup`).

Live admin updates to `allowed_commands` follow the same rules: `*` is rejected unless `EXECUTOR_ALLOW_WILDCARD=1` is set in the process environment.

## Injection hardening

- Payloads are **not** passed to `sh -c`. The command is split into argv and executed directly.
- Shell metacharacters (`;|&$()`\`<>` and newlines) are rejected outright.
- An allow entry of `echo` does **not** match `echohi` or `echo hi; rm -rf /`.

## Timeouts

`Execute` honors the incoming gRPC/mesh context deadline. When none is set (worker-pool dispatch uses a long-lived loop context), the module applies **`EXECUTOR_TIMEOUT`** (default **5m**, overridable via settings `timeout`). When the deadline fires, the child process is killed.

## Output bounds

Stdout and stderr are capped at **`EXECUTOR_MAX_OUTPUT`** (default **1 MiB**). Overflow fails closed with `output limit exceeded` — unbounded buffers are not used.

## Logging

Rejected commands and allowlist configuration are logged with truncated previews and SHA-256 fingerprints, not full secrets or payloads.

## Network / secrets

- Do not allow tokens that can exfiltrate secrets (`curl`, `wget`, `scp`, broad `bash`/`sh`) unless the host is fully trusted and isolated.
- Run the module under a least-privilege OS user with no write access to MuxCore data dirs beyond what the task needs.

## Wildcard flag

| Variable | Purpose |
|----------|---------|
| `EXECUTOR_ALLOW_WILDCARD` | Must be `1` / `true` / `yes` / `on` for `*` in env or live settings |

Without this flag, `*` in the allowlist is ignored (env) or rejected (settings update).
