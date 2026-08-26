# ServerPanel — build & verification
# POSIX-sh compatible (Linux CI + Git Bash/MSYS on Windows).
# All Node.js workloads run inside Linux containers so behavior matches CI and
# hosts are isolated from local antivirus interference with node_modules.
SHELL := /bin/sh

GO ?= go
GOFMT ?= gofmt
DOCKER ?= docker

BIN_DIR := bin
TOOLS_BIN := $(abspath $(BIN_DIR)/tools)
PKG_DIR := dist/package

NODE_IMAGE ?= node:24-bookworm-slim
PW_IMAGE ?= mcr.microsoft.com/playwright:v1.62.1-jammy

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/dursuntokgoz/OpenControl/internal/core.buildVersion=$(VERSION) \
	-X github.com/dursuntokgoz/OpenControl/internal/core.buildCommit=$(COMMIT) \
	-X github.com/dursuntokgoz/OpenControl/internal/core.buildBuildDate=$(BUILD_DATE)

# Host detection: race detector needs cgo/gcc (absent on this Windows host);
# MSYS needs path-conversion disabled for docker arguments.
HOST_IS_MSYS := $(shell uname -o 2>/dev/null | grep -i msys)
ifeq ($(HOST_IS_MSYS),)
	RACE := -race
else
	RACE :=
endif
export MSYS_NO_PATHCONV := 1

WEB_RUN := $(DOCKER) run --rm -v "$(CURDIR):/w" -w /w/web \
	-v serverpanel-web-node-modules:/w/web/node_modules $(NODE_IMAGE)
E2E_RUN := $(DOCKER) run --rm -v "$(CURDIR):/w" -w /w \
	-v serverpanel-e2e-node-modules:/w/test/e2e/node_modules $(PW_IMAGE)

GOLANGCI_LINT_VERSION := v2.13.1

.PHONY: help deps vendor-go lint fmt-fix test cover build build-linux build-web up down itest logs-itest e2e e2e-deps smoke clean verify

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

deps: ## Install all dependencies (Go modules, containers, lint tools)
	$(GO) mod download
	$(MAKE) vendor-go
	$(DOCKER) pull $(NODE_IMAGE)
	$(DOCKER) pull $(PW_IMAGE)
	GOBIN=$(TOOLS_BIN) $(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

vendor-go: ## Refresh vendor/ directory (hermetic container builds)
	$(GO) mod vendor

lint: ## golangci-lint + gofmt check + eslint + tsc (JS inside containers)
	@unformatted="$$($(GOFMT) -l cmd internal)"; \
	if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi
	$(TOOLS_BIN)/golangci-lint run
	$(WEB_RUN) sh -c "npm ci --no-fund --no-audit && npm run lint && npx tsc --noEmit"

fmt-fix: ## Apply gofmt fixes
	$(GO) fmt ./...

test: ## Unit tests (Go + frontend Vitest) with coverage
	$(GO) test ./... $(RACE) -covermode=atomic -coverprofile=coverage.out
	$(GO) tool cover -func=coverage.out | tail -1
	$(WEB_RUN) sh -c "npm ci --no-fund --no-audit && npm test"

cover: ## Open HTML coverage report
	$(GO) tool cover -html=coverage.out

build: ## Build host binaries + production frontend bundle (containers)
	$(GO) build -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/panel-api ./cmd/panel-api
	$(GO) build -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/panel-agent ./cmd/panel-agent
	$(GO) build -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/panelctl ./cmd/panelctl
	$(MAKE) build-web

build-web: ## Production frontend bundle via containerized toolchain
	$(WEB_RUN) sh -c "npm ci --no-fund --no-audit && npm run build"

build-linux: ## Cross-compile Linux artifacts into dist/package
	rm -rf $(PKG_DIR)
	mkdir -p $(PKG_DIR)/bin $(PKG_DIR)/systemd $(PKG_DIR)/web
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags '$(LDFLAGS)' -o $(PKG_DIR)/bin/panel-api ./cmd/panel-api
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags '$(LDFLAGS)' -o $(PKG_DIR)/bin/panel-agent ./cmd/panel-agent
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags '$(LDFLAGS)' -o $(PKG_DIR)/bin/panelctl ./cmd/panelctl
	cp packaging/systemd/*.service $(PKG_DIR)/systemd/
	cp packaging/install.sh $(PKG_DIR)/install.sh
	chmod +x $(PKG_DIR)/install.sh $(PKG_DIR)/bin/*

up: ## Bring up the systemd-enabled integration test environment
	$(DOCKER) compose -f docker-compose.test.yml up -d --build

down: ## Tear down the integration test environment
	$(DOCKER) compose -f docker-compose.test.yml down -v --remove-orphans

itest: ## Integration tests inside systemd containers
	$(MAKE) up
	@echo "waiting for systemd..."
	@ok=""; \
	for i in $$(seq 1 60); do \
		state=$$($(DOCKER) compose -f docker-compose.test.yml exec -T target systemctl is-system-running 2>/dev/null || true); \
		if [ "$$state" = "running" ] || [ "$$state" = "degraded" ]; then ok=1; break; fi; \
		sleep 2; \
	done; \
	[ -n "$$ok" ] || { echo "systemd did not become ready"; $(MAKE) logs-itest; exit 1; }
	$(DOCKER) compose -f docker-compose.test.yml exec -T target /usr/local/bin/run-itest.sh; \
	status=$$?; \
	if [ "$$status" -ne 0 ]; then $(MAKE) logs-itest; fi; \
	exit $$status

logs-itest: ## Dump integration container logs (diagnostics)
	$(DOCKER) compose -f docker-compose.test.yml logs --no-color target || true

e2e-deps: ## Pull E2E container images
	$(DOCKER) pull $(PW_IMAGE)
	$(DOCKER) pull $(NODE_IMAGE)

e2e: ## Playwright E2E against real panel-api (all inside container)
	$(MAKE) build-linux
	$(MAKE) build-web
	cp -r web/dist/. $(PKG_DIR)/web/
	$(E2E_RUN) sh -ec '\
		npm ci --prefix test/e2e --no-fund --no-audit; \
		mkdir -p /tmp/sp-data; \
		SERVERPANEL_HTTP_LISTEN=127.0.0.1:8117 \
		SERVERPANEL_DB_SQLITE_PATH=/tmp/sp-data/e2e.db \
		SERVERPANEL_WEB_DIST=/w/web/dist \
		/w/dist/package/bin/panel-api & \
		api_pid=$$!; \
		trap "kill $$api_pid 2>/dev/null || true" EXIT INT TERM; \
		ok=""; \
		for i in $$(seq 1 30); do \
			if curl -fsS http://127.0.0.1:8117/healthz >/dev/null 2>&1; then ok=1; break; fi; \
			sleep 1; \
		done; \
		[ -n "$$ok" ] || { echo "panel-api did not become healthy"; exit 1; }; \
		BASE_URL=http://127.0.0.1:8117 npx --prefix test/e2e playwright test --config test/e2e/playwright.config.ts'

smoke: ## Clean-container install.sh smoke test (Debian + systemd)
	$(MAKE) build-linux
	$(MAKE) build-web
	cp -r web/dist/. $(PKG_DIR)/web/
	$(DOCKER) build -q -t serverpanel/smoke-base:latest -f packaging/docker/Dockerfile.systemd-debian packaging/docker
	@cid="sp-smoke-$$(date +%s)"; \
	cleanup() { $(DOCKER) rm -f "$$cid" >/dev/null 2>&1 || true; }; \
	trap cleanup EXIT INT TERM; \
	$(DOCKER) run -d --name "$$cid" --hostname panel-smoke --privileged --cgroupns=host \
		-v "$(CURDIR)/$(PKG_DIR):/pkg:ro" \
		serverpanel/smoke-base:latest >/dev/null; \
	ok=""; \
	for i in $$(seq 1 60); do \
		state=$$($(DOCKER) exec "$$cid" systemctl is-system-running 2>/dev/null || true); \
		if [ "$$state" = "running" ] || [ "$$state" = "degraded" ]; then ok=1; break; fi; \
		sleep 2; \
	done; \
	if [ -z "$$ok" ]; then echo "systemd did not start in smoke container"; exit 1; fi; \
	$(DOCKER) exec "$$cid" /bin/sh -ec '/pkg/install.sh /pkg; \
		installed=""; \
		for i in $$(seq 1 40); do \
			if curl -fsS http://127.0.0.1:8080/healthz; then installed=1; break; fi; \
			sleep 1; \
		done; \
		if [ -z "$$installed" ]; then \
			journalctl -u panel-api --no-pager | tail -50; \
			systemctl status panel-api --no-pager || true; \
			exit 1; \
		fi; \
		panelctl doctor -config /etc/serverpanel/config.yaml; \
		echo SMOKE_OK'; \
	status=$$?; cleanup; exit $$status

verify: ## MAIN GATE: lint + test + build + itest + e2e + smoke
	$(MAKE) lint
	$(MAKE) test
	$(MAKE) build
	$(MAKE) itest
	$(MAKE) e2e
	$(MAKE) smoke

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) $(WEB_DIR)/dist $(PKG_DIR) coverage.out
