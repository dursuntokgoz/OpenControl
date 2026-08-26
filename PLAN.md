# ServerPanel — Plan

Open-source Linux server control panel (cPanel/WHM functionally referenced, fully original code/UI).
See `DECISIONS.md` for architecture rationale and `TASKS.md` for live task status.

## Architecture Overview

```
┌─────────────┐  HTTPS   ┌──────────────────┐  Unix socket (JSON-RPC)  ┌───────────────┐
│  Browser     │ ───────▶ │  panel-api       │ ───────────────────────▶ │  panel-agent  │
│  /admin      │          │  (unprivileged)  │                          │  (root)       │
│  / (user)    │          │  Go + chi        │                          │  whitelist    │
└─────────────┘          │  SQLite/Postgres │                          └──────┬────────┘
                         └──────────────────┘                                 │ exec
                              ▲                                               ▼
                         React SPA (web/)                        nginx/apache/php/mariadb/
                                                                bind/pdns/postfix/dovecot/
                                                                nftables/firewalld/acme
```

- **panel-api**: HTTP API + serves built React assets. Runs as `panel:panel`. Never root.
- **panel-agent**: privileged helper. Root. Listens on `/run/serverpanel/agent.sock`.
  Executes only whitelisted operations; every request authenticated via socket peer
  credentials (SO_PEERCRED) + shared secret. Argument arrays only, never shell strings.
- **panelctl**: CLI for install/bootstrap/diagnostics.
- **Job queue**: DB-backed persistent queue with workers inside panel-api.

## Phases

| Phase | Content | Acceptance |
|-------|---------|------------|
| 0 | Repo skeleton, Makefile, CI, lint, Docker test env | `make verify` green |
| 1 | Auth, RBAC, account CRUD, audit log, DB migrations | Integration tests pass, TOTP 2FA works |
| 2 | panel-agent + whitelist + system info + service mgmt | nginx restart via agent test passes |
| 3 | Web server + domains + PHP-FPM pools + PHP selector | Test domain returns HTTP 200 |
| 4 | File manager + FTP/SFTP + quota | Path traversal & quota tests pass |
| 5 | DNS zone management + editor | `dig @localhost` returns record |
| 6 | SSL/Let's Encrypt (pebble) + AutoSSL job | Cert from test CA issued & renewed |
| 7 | Database management (MySQL/PostgreSQL) | DB+user created, connection verified |
| 8 | Mail stack (Postfix/Dovecot) | SMTP send + IMAP login tests pass |
| 9 | Cron, statistics, log viewer, resource graphs | Tests pass |
| 10 | Backup/restore (local+S3/SFTP), scheduled jobs | Backup→delete→restore roundtrip passes |
| 11 | Security module (firewall, fail2ban), packages | Tests pass |
| 12 | Resellers, i18n, themes, self-update, packaging | Clean-container install smoke passes |
| 13 | Docs, security hardening review, performance | docs complete, verify + e2e green |

## Verification Gates

`make verify` = lint + test + build + itest + e2e + smoke. Exit code must be 0.
