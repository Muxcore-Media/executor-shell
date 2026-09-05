# Changelog

## [0.1.4] — 2026-08-10

### Added

- Advertise `settings` capability so admin-ui discovers SettingsProvider without ListAll probing.

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0/).

## [Unreleased]

### Security

- Enable TLS on the module gRPC listener by default via `internal/grpctls`; plaintext only when `MUXCORE_INSECURE_DISABLE_TLS=true` (or `MUXCORE_GRPC_INSECURE`).

### Changed

- Default gRPC listen address is `127.0.0.1:9602` (override with `EXECUTOR_GRPC_ADDR`)

## [0.1.3] — 2026-08-10

### Added

- `RegisterSettings` / `SettingsUpdater` for live `allowed_commands` (`EXECUTOR_ALLOWED_COMMANDS`)
- Pin `core` / contracts / `sdk/go/module` to **v0.5.2**

## [0.1.2] — 2026-08-10

### Fixed
- Sync Info()/muxcore.json version to **0.1.2**.

## v0.1.1 (2026-08-10)

- SECURITY.md: forbid `*` allowlist in production; timeout/cancel guidance
- Tests: Execute context timeout and cancel kill the process promptly

## v0.1.0 (2026-08-09)

- Initial shell executor with command allowlist
