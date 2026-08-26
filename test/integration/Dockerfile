# Integration test target: Debian 12 with systemd, panel binaries built from
# source, and the services the panel will manage in later phases.
FROM golang:1.26 AS build
WORKDIR /src
ENV GOFLAGS=-mod=vendor GOPROXY=off
COPY go.mod go.sum ./
COPY vendor ./vendor
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -o /out/panel-api ./cmd/panel-api \
 && CGO_ENABLED=0 go build -o /out/panel-agent ./cmd/panel-agent \
 && CGO_ENABLED=0 go build -o /out/panelctl ./cmd/panelctl

FROM debian:12
RUN apt-get update && apt-get install -y --no-install-recommends \
        systemd systemd-sysv dbus curl ca-certificates nginx sqlite3 procps \
    && rm -rf /var/lib/apt/lists/*

COPY --from=build /out/panel-api /usr/local/bin/panel-api
COPY --from=build /out/panel-agent /usr/local/bin/panel-agent
COPY --from=build /out/panelctl /usr/local/bin/panelctl
COPY packaging/systemd/panel-api.service /etc/systemd/system/panel-api.service
COPY packaging/systemd/panel-agent.service /etc/systemd/system/panel-agent.service

# Test-only configuration; tokens here are fixtures, not production secrets.
RUN useradd -r -s /usr/sbin/nologin panel \
 && mkdir -p /etc/serverpanel /var/lib/serverpanel /usr/share/serverpanel/web /run/serverpanel \
 && chown -R panel:panel /var/lib/serverpanel \
 && printf 'http:\n  listen: 127.0.0.1:8080\n  webDist: ""\ndatabase:\n  driver: sqlite\n  sqlitePath: /var/lib/serverpanel/panel.db\nagent:\n  socketPath: /run/serverpanel/agent.sock\n  token: itest-token-not-secret-by-design\nlog:\n  format: text\n' > /etc/serverpanel/config.yaml \
 && systemctl enable panel-api.service panel-agent.service

COPY test/integration/scripts/run-itest.sh /usr/local/bin/run-itest.sh
RUN chmod +x /usr/local/bin/run-itest.sh

STOPSIGNAL SIGRTMIN+3
CMD ["/sbin/init"]
