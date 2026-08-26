# Errors

Encountered error → root cause → resolution. Keep entries short; link full output where useful.

| Date | Phase | Error | Root cause | Resolution |
|------|-------|-------|-----------|------------|
| 2026-08-26 | setup | `winget install GoLang.Go` failed: msstore source cert mismatch (0x8a15005e) | msstore endpoint TLS interception | Retry with `--source winget` — success |
| 2026-08-26 | setup | `choco install make`: access denied to C:\ProgramData\chocolatey\lib-bad | Non-elevated shell; choco dirs require admin | Used `winget install ezwinports.make` instead |
| 2026-08-26 | setup | Fresh shells cannot find go/node/make after install | Parent process env captured before PATH change | Prepend refreshed Machine+User PATH (and Git usr/bin for sh) in every shell invocation |
| 2026-08-26 | p0 | config tests fail on Windows: `/run/serverpanel/agent.sock` judged not absolute | `filepath.IsAbs` is host-OS specific | Validate POSIX-absoluteness explicitly (`strings.HasPrefix(p,"/")`) — product targets Linux |
| 2026-08-26 | p0 | node.exe + playwright cli.js repeatedly deleted from disk (MSI & zip locations) | Kaspersky Endpoint Security 12.10 quarantine of unsigned payloads | Run ALL Node workloads inside Docker containers; node_modules in named volumes |
| 2026-08-26 | p0 | `go mod download` inside Dockerfile fails: x509 unknown authority | Corporate TLS-intercepting proxy untrusted inside container | Vendor dependencies (`go mod vendor`), build with GOFLAGS=-mod=vendor GOPROXY=off |
| 2026-08-26 | p0 | docker exec path mangled to `C:/Program Files/Git/usr/local/bin/run-itest.sh` | MSYS path conversion of POSIX-looking args | `export MSYS_NO_PATHCONV := 1` in Makefile |
| 2026-08-26 | p0 | smoke: "System has not been booted with systemd as init system" | `docker run <image> /bin/sh -c ...` replaced PID 1 | Start container detached with default CMD (/sbin/init), then `docker exec` the test script |
| 2026-08-26 | p0 | smoke container name contained literal `(date +%s)` | Over-escaped `$$$$$(...)` in make recipe | Use `$$(date +%s)` (make→shell single-dollar) |
