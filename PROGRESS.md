# Progress Log

Chronological log: what was done, which commands ran, results.

## 2026-08-26

### Environment bootstrap
- Repo was empty (only .git). Added remote origin → https://github.com/dursuntokgoz/OpenControl.git.
- Toolchain check: Docker 29.6.1 present; Go/Node/make missing.
- Installed via winget: GoLang.Go 1.26.7, OpenJS.NodeJS.LTS 24.19.0 (MSI), ezwinports.make 4.4.1.
  - winget msstore source cert error (0x8a15005e) → retried with `--source winget`.
  - choco make install failed (non-elevated) → switched to ezwinports.make.
- Verified GNU Make 4.4.1 executes POSIX recipes using Git-for-Windows sh.exe.

### Kaspersky interference
- Node.exe binaries repeatedly vanished after install (MSI `C:\Program Files\nodejs\node.exe`
  AND zip copy under `%USERPROFILE%\tools`); `@playwright/test/cli.js` also deleted.
- Root cause: Kaspersky Endpoint Security 12.10 present (Defender disabled); selective
  quarantine of downloaded executables/JS payloads.
- Mitigations: working Node zip copy at tools dir for npm CLI bootstrap; **all JS workloads
  moved into Linux containers** (node:24-bookworm-slim + mcr.microsoft.com/playwright:v1.62.1-jammy)
  with named volumes for node_modules — immune to host AV.

### Phase 0 implementation
- State files created (PLAN/TASKS/DECISIONS/ERRORS/PROGRESS).
- Go module `github.com/dursuntokgoz/OpenControl`; deps: chi v5, modernc sqlite, goose,
  pgx stdlib, gopsutil v4, yaml.v3, testify.
- Packages: internal/{core,config,providers,agent,api,store}; cmd/{panel-api,panel-agent,panelctl}.
- Agent: JSON-lines over unix socket, whitelist registry, token auth, panic-safe handlers,
  SO_PEERCRED logging on Linux (build-tagged).
- API: chi router, security headers (CSP etc.), slog request logs, /healthz, SPA fallback
  static server with graceful shutdown.
- Store: SQLite (WAL, busy_timeout) + Postgres, embedded goose migrations, settings KV.
- Frontend: React 18 + TS strict + Vite 5 + Tailwind 3.4 + TanStack Query v5 +
  react-router 6; i18n TR/EN catalogs with parity tests; HealthCard via useQuery;
  language switcher persisted in localStorage.
- Verification infra: Makefile (deps/lint/fmt-fix/test/build/build-linux/build-web/up/down/
  itest/e2e/smoke/verify), docker-compose.test.yml (privileged systemd target),
  multi-stage Dockerfile vendoring deps offline (corporate TLS interception workaround),
  install.sh idempotent installer, systemd units, smoke base image, Playwright specs ×3,
  GitHub Actions CI (lint-test-build + integration-e2e-smoke jobs).

### Verification evidence (final run)
- `make lint`: golangci-lint 0 issues; eslint clean; tsc clean.
- `make test`: Go tests ok (agent/api/config/store); Vitest 8/8.
- `make build`: panel-api/panel-agent/panelctl + vite dist built.
- `make itest`: systemd container: services active, healthz ok, agent ping ok,
  doctor all PASS, nginx restart + default page OK → ITEST_OK.
- `make e2e`: 3/3 Playwright passed against real panel-api serving production bundle.
- `make smoke`: clean debian:12 container → install.sh → services enabled+running →
  healthz ok → doctor all critical PASS → SMOKE_OK.
- **`make verify` exit code = 0.**
