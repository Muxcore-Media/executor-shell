# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Shell command executor gRPC service (executor.v1)
- `CanHandle` returns true for task type `"shell"`
- `Execute` runs payload as shell command via `sh -c`; Result is stdout, Error is stderr (or exec error) on failure
- Security: optional allow-list via `EXECUTOR_ALLOWED_COMMANDS` env var
- Contract declaration: `Executor` with `MinCoreVersion: 0.4.0`
- Full test suite: module info, CanHandle, module lifecycle
- Production deployment: Dockerfile, docker-compose, systemd unit
- Configuration via env vars: `EXECUTOR_GRPC_ADDR`, `EXECUTOR_ALLOWED_COMMANDS`

## [0.1.0] - 2026-06-13

### Added

- Initial project scaffold
