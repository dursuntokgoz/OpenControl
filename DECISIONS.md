# Decisions

Log of significant architecture decisions. Newest first within sections.

## ADR-0001: Pure-Go SQLite driver (modernc.org/sqlite)
- **Status**: accepted
- **Context**: Default DB is SQLite; we also ship .deb/.rpm and cross-compile.
- **Decision**: Use `modernc.org/sqlite` (CGO-free) instead of `mattn/go-sqlite3`.
- **Consequences**: Single static binary, trivial cross-compilation, slightly lower raw
  performance than CGO builds — acceptable for control-plane workloads.

## ADR-0002: Privilege separation via panel-agent
- **Status**: accepted
- **Context**: Web panel must never run as root.
- **Decision**: Separate root process `panel-agent` listening on a Unix domain socket
  (`/run/serverpanel/agent.sock`). API connects as unprivileged user. Agent validates
  each request against a strict operation whitelist with typed arguments
  (no free-form commands). Socket peer creds are verified.
- **Consequences**: All privileged mutations funnel through auditable choke point;
  attack surface limited to whitelist grammar.

## ADR-0003: No shell string interpolation, ever
- **Status**: accepted
- **Decision**: Every external process invocation uses `exec.Command(name, args...)`
  with argument arrays built from validated structs. User input never reaches a shell.
- **Enforcement**: Code review + lint rule + dedicated security tests (Phase 4+).

## ADR-0004: Config generation via templates + syntax check + atomic rollback
- **Status**: accepted
- **Decision**: All service configs are rendered from Go text/template sources into a
  staging dir, syntax-checked (`nginx -t`, `named-checkzone`, `postfix check`, ...),
  then moved into place atomically; previous files kept for rollback on failure.

## ADR-0005: Frontend stack pinned versions
- **Status**: accepted
- **Decision**: React 18.3 + TypeScript 5.x + Vite 5.x + Tailwind CSS 3.4 (not v4:
  different tooling pipeline, keep stable PostCSS flow) + TanStack Query v5.
  ESLint 9 flat config.

## ADR-0006: Migrations with goose
- **Status**: accepted
- **Decision**: `pressly/goose/v3` embedded SQL migrations supporting both SQLite and
  PostgreSQL dialects where possible; store layer abstracts dialect differences.

## ADR-0007: Windows dev host support
- **Status**: accepted
- **Context**: Primary development machine is Windows with Docker Desktop; CI is Linux.
- **Decision**: Makefile recipes are POSIX-sh compatible (make resolves sh.exe from Git
  for Windows). Linux-only verification (itest/e2e/smoke) executes inside containers so
  behavior matches production targets (Debian/Ubuntu/AlmaLinux).

## ADR-0008: Job queue in database, not Redis
- **Status**: accepted
- **Decision**: Persistent jobs table with `SELECT ... ` claim pattern + worker goroutines.
  Survives restarts without extra infrastructure; matches "SQLite default" constraint.

## ADR-0009: Session auth: cookie + argon2id; TOTP RFC 6238; scoped API tokens
- **Status**: accepted
- **Decision**: Server-side sessions in DB, HttpOnly+Secure+SameSite=Lax cookies,
  argon2id password hashing (RFC 9106 recommended params), TOTP enrollment per user,
  API tokens with explicit scopes stored hashed.

## ADR-0010: i18n TR+EN from day one
- **Status**: accepted
- **Decision**: Frontend message catalogs (TypeScript modules), backend error codes
  translated client-side; UI language switchable per-user.

## ADR-0011: All Node.js workloads run in Linux containers
- **Status**: accepted
- **Context**: Dev host runs Kaspersky Endpoint Security which quarantines
  node.exe/playwright payloads nondeterministically; corporate proxy breaks TLS inside
  Docker builds.
- **Decision**: eslint/tsc/vitest/vite build and Playwright E2E execute inside pinned
  containers (`node:24-bookworm-slim`, `mcr.microsoft.com/playwright:v1.62.1-jammy`);
  node_modules live in named volumes, never on the host FS. Go dependencies are vendored
  so container image builds are offline/hermetic (GOFLAGS=-mod=vendor GOPROXY=off).
- **Consequences**: Identical JS toolchain local/CI; immune to host AV; slightly heavier
  first-run (image pulls).

## ADR-0012: Vendor directory committed for hermetic builds
- **Status**: accepted
- **Decision**: `go mod vendor` output is committed and used by all container-based Go
  builds. CI still runs `go mod download` for the lint/test path.
- **Consequences**: Reproducible integration/smoke images even behind TLS interception;
  larger repo (~15 MB).
