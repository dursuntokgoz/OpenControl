# Tasks

Status: `todo` / `doing` / `blocked` / `done`

## Phase 0 — Skeleton & Verification Infrastructure  ✅ COMPLETE (verify exit=0)
| ID | Title | Status | Acceptance |
|----|-------|--------|------------|
| P0-01 | State files (PLAN/TASKS/PROGRESS/DECISIONS/ERRORS) | done | Files exist and are accurate |
| P0-02 | Go module + cmd/panel-api,panel-agent,panelctl compile | done | `go build ./...` succeeds |
| P0-03 | internal/core, internal/providers, internal/store packages with interfaces | done | Packages compile, interfaces documented |
| P0-04 | Frontend skeleton React18+TS+Vite+Tailwind+TanStack Query | done | `npm run build` produces dist |
| P0-05 | Makefile targets deps/lint/test/build/up/itest/e2e/smoke/verify | done | Each target runs successfully |
| P0-06 | golangci-lint + eslint + tsc configs | done | Lint passes clean (0 issues) |
| P0-07 | Unit test skeletons (Go + Vitest) with real assertions | done | `make test` green (Go ok, Vitest 8/8) |
| P0-08 | Docker compose test env (systemd container) | done | `docker compose up` works, systemd running |
| P0-09 | Integration test harness (itest) | done | itest: services active, doctor PASS, ITEST_OK |
| P0-10 | Playwright E2E skeleton | done | `make e2e` green (3/3 specs) |
| P0-11 | packaging/systemd units + install.sh v1 + smoke target | done | smoke: clean debian:12 install → SMOKE_OK |
| P0-12 | GitHub Actions CI workflow | done | Workflow mirrors verify chain |

## Phase 1 — Auth, RBAC, Accounts, Audit  ✅ COMPLETE (verify exit=0, commit cd26140)
| ID | Title | Status | Acceptance |
|----|-------|--------|------------|
| P1-01 | Store layer: open SQLite/Postgres, goose migrations | done | Migration up/down tests pass |
| P1-02 | Domain models: User, Account, Package, Session, AuditLog | done | Models + validation tests |
| P1-03 | argon2id password hashing service | done | Hash/verify + vectors tests |
| P1-04 | Session management (create/rotate/expire) + secure cookies | done | Unit + integration tests |
| P1-05 | TOTP 2FA enrollment + verification | done | RFC vectors test passes |
| P1-06 | RBAC middleware admin/reseller/user | done | Authorization matrix tests |
| P1-07 | Account CRUD API + handlers | done | Integration tests via HTTP |
| P1-08 | Audit log writes on every mutation | done | Audit assertions in tests |
| P1-09 | Rate limiting + CSRF + security headers middleware | done | Middleware tests |
| P1-10 | Bootstrap admin (env + panelctl bootstrap-admin + install.sh) | done | E2E login as bootstrapped admin |
| P1-11 | Web login flow (LoginPage, AuthProvider, route guards) | done | E2E 7/7 incl. redirect + wrong-password |
| P1-12 | UI screenshots for docs | done | docs/screenshots/01-05.png in repo |
