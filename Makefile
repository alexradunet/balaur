PREFIX ?= $(HOME)/.local
BALAUR_BIN_DIR ?= $(PREFIX)/bin
BALAUR_CONFIG_DIR ?= $(HOME)/.config/balaur
SYSTEMD_USER_DIR ?= $(HOME)/.config/systemd/user
SERVICE_NAME ?= balaur
# Prod data dir (matches the systemd unit default). Override if your env file
# points BALAUR_DATA_DIR elsewhere: make promote BALAUR_DATA_DIR=/path/to/pb_data
BALAUR_DATA_DIR ?= $(HOME)/.local/share/balaur/pb_data

.PHONY: help tools dev dev-port-open dev-port-close run seed build install-user-service start-user-service stop-user-service restart-user-service promote status-user-service logs-user-service test race fmt vet staticcheck vulncheck lint

help:
	@echo "make tools  # install dlv + air, activate the pre-commit lint hook"
	@echo "make dev    # hot-reload staging via air; opens dev :$(DEV_PORT) on $(NETBIRD_IFACE) while running, closes on exit"
	@echo "make dev-port-open   # open dev :$(DEV_PORT) over NetBird by hand (sudo ufw)"
	@echo "make dev-port-close  # close dev :$(DEV_PORT) again (sudo ufw)"
	@echo "make run    # start Balaur once (go run . serve)"
	@echo "make build  # build a CGO-free binary"
	@echo "make install-user-service   # install binary + systemd user service"
	@echo "make start-user-service     # enable and start the user service"
	@echo "make stop-user-service      # stop the user service"
	@echo "make restart-user-service   # rebuild, reinstall, and restart the service"
	@echo "make promote                # staging->prod: clean tree + tests + data snapshot + restart"
	@echo "make status-user-service    # show the service status"
	@echo "make logs-user-service      # follow the service logs"
	@echo "make test   # go test ./..."
	@echo "make race   # go test -race ./... (requires CGO)"
	@echo "make fmt    # gofmt -l ."
	@echo "make vet    # go vet ./..."
	@echo "make staticcheck  # honnef.co staticcheck ./..."
	@echo "make vulncheck    # govulncheck ./..."
	@echo "make lint   # fmt + vet + staticcheck + test"

# Hosts the dev server accepts beyond loopback (guardLocalUI in
# internal/web/web.go 403s everything else). air's [build].env doesn't
# propagate to the child binary, so we export it here — the built process
# inherits it. Override on the CLI: make dev BALAUR_ALLOWED_HOSTS=1.2.3.4
BALAUR_ALLOWED_HOSTS ?= 192.168.50.12,100.124.242.131,balaur,balaur.local,balaur-113-87.netbird.cloud,100.124.113.87
export BALAUR_ALLOWED_HOSTS

# The dev/staging instance binds 0.0.0.0:$(DEV_PORT), but unlike prod (8080) its
# mesh port is NOT left permanently open (see dev_env/debian). `make dev` opens it
# on the NetBird interface while it runs and closes it on exit; both steps are
# best-effort so `make dev` still works off the mesh box (e.g. local-only). Open
# or close it by hand with `make dev-port-open` / `make dev-port-close`.
NETBIRD_IFACE ?= wt0
DEV_PORT ?= 8090

# Reproducible dev-tool setup. dlv powers the VS Code "Debug: balaur serve"
# launch config; air drives `make dev`. Both land in $(go env GOPATH)/bin —
# ensure that's on your PATH. Re-run any time to update to the latest.
# Also points git at the committed .githooks/ so the pre-commit lint hook is
# active without copying anything.
tools:
	go install github.com/go-delve/delve/cmd/dlv@latest
	go install github.com/air-verse/air@latest
	git config core.hooksPath .githooks
	@echo "dev tools installed; pre-commit hook active (.githooks/pre-commit)"

dev:
	@$(MAKE) --no-print-directory dev-port-open || echo "dev: :$(DEV_PORT) not opened automatically — run 'make dev-port-open' (sudo) or open it manually"
	@trap '$(MAKE) --no-print-directory dev-port-close || true' EXIT; \
	if [ -f dev.env ]; then set -a; . ./dev.env; set +a; echo "dev: sourced dev.env (BALAUR_MISTRAL_KEY set: $${BALAUR_MISTRAL_KEY:+yes})"; fi; \
	if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		go run github.com/air-verse/air@latest; \
	fi

# Open/close the dev/staging port on the NetBird interface on demand. Opening a
# firewall port needs root, hence sudo. Closing ignores "no such rule" so it is
# safe to run when the port is already closed.
dev-port-open:
	sudo ufw allow in on $(NETBIRD_IFACE) to any port $(DEV_PORT) proto tcp comment 'Balaur dev/staging (temporary; opened by make dev)'

dev-port-close:
	-sudo ufw delete allow in on $(NETBIRD_IFACE) to any port $(DEV_PORT) proto tcp

run:
	go run . serve

seed:
	go run . seed $(ARGS)

build:
	CGO_ENABLED=0 go build -o balaur .

install-user-service: build
	mkdir -p "$(BALAUR_BIN_DIR)" "$(BALAUR_CONFIG_DIR)" "$(BALAUR_DATA_DIR)" "$(SYSTEMD_USER_DIR)"
	install -m 0755 balaur "$(BALAUR_BIN_DIR)/balaur"
	install -m 0644 contrib/systemd/balaur.service "$(SYSTEMD_USER_DIR)/$(SERVICE_NAME).service"
	@if [ ! -f "$(BALAUR_CONFIG_DIR)/env" ]; then \
		install -m 0600 contrib/systemd/balaur.env.example "$(BALAUR_CONFIG_DIR)/env"; \
	fi
	systemctl --user daemon-reload

start-user-service:
	systemctl --user enable --now "$(SERVICE_NAME)"

stop-user-service:
	systemctl --user stop "$(SERVICE_NAME)"

restart-user-service: install-user-service
	systemctl --user restart "$(SERVICE_NAME)"

# Promote the current checkout (what you validated on staging via `make dev`)
# to the prod service. The checkout is shared, so refuse a dirty tree; run the
# suite; build first so a compile failure leaves prod untouched; snapshot prod
# data while the service is stopped — a clean pre-migration restore point, since
# pending migrations apply on the next start; then reinstall + restart. Prod is
# down for the snapshot + restart window (seconds, plus the data copy). Backups
# accumulate as pb_data.bak-* — prune them yourself.
promote:
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "promote: working tree is dirty — commit or stash first:"; \
		git status --short; \
		exit 1; \
	fi
	@echo "promote: tree clean — running tests…"
	$(MAKE) test
	@echo "promote: building (a failure here leaves prod on the old binary)…"
	$(MAKE) build
	@echo "promote: stopping prod + snapshotting data…"
	-systemctl --user stop "$(SERVICE_NAME)"
	@if [ -d "$(BALAUR_DATA_DIR)" ]; then \
		snap="$(BALAUR_DATA_DIR).bak-$$(date +%Y%m%d-%H%M%S)"; \
		cp -a "$(BALAUR_DATA_DIR)" "$$snap"; \
		echo "promote: snapshot -> $$snap"; \
	else \
		echo "promote: no data dir at $(BALAUR_DATA_DIR) — skipping snapshot"; \
	fi
	$(MAKE) install-user-service
	systemctl --user enable "$(SERVICE_NAME)"
	systemctl --user restart "$(SERVICE_NAME)"
	@echo "promote: done — verify with 'make status-user-service'"

status-user-service:
	systemctl --user status "$(SERVICE_NAME)"

logs-user-service:
	journalctl --user -u "$(SERVICE_NAME)" -f

test:
	go test ./...

race:
	CGO_ENABLED=1 go test -race ./...

fmt:
	@unformatted="$$(gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "These files need gofmt -w:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

vet:
	go vet ./...

staticcheck:
	go run honnef.co/go/tools/cmd/staticcheck@latest ./...

vulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

lint: fmt vet staticcheck test
