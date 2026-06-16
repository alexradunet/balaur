PREFIX ?= $(HOME)/.local
BALAUR_BIN_DIR ?= $(PREFIX)/bin
BALAUR_CONFIG_DIR ?= $(HOME)/.config/balaur
SYSTEMD_USER_DIR ?= $(HOME)/.config/systemd/user
SERVICE_NAME ?= balaur

.PHONY: help tools dev run seed build install-user-service start-user-service stop-user-service restart-user-service status-user-service logs-user-service test race fmt vet lint

help:
	@echo "make tools  # install dlv + air, activate the pre-commit lint hook"
	@echo "make dev    # start Balaur with hot reload via air"
	@echo "make run    # start Balaur once (go run . serve)"
	@echo "make build  # build a CGO-free binary"
	@echo "make install-user-service   # install binary + systemd user service"
	@echo "make start-user-service     # enable and start the user service"
	@echo "make stop-user-service      # stop the user service"
	@echo "make restart-user-service   # rebuild, reinstall, and restart the service"
	@echo "make status-user-service    # show the service status"
	@echo "make logs-user-service      # follow the service logs"
	@echo "make test   # go test ./..."
	@echo "make race   # go test -race ./... (requires CGO)"
	@echo "make fmt    # gofmt -l ."
	@echo "make vet    # go vet ./..."
	@echo "make lint   # fmt + vet + test"

# Hosts the dev server accepts beyond loopback (guardLocalUI in
# internal/web/web.go 403s everything else). air's [build].env doesn't
# propagate to the child binary, so we export it here — the built process
# inherits it. Override on the CLI: make dev BALAUR_ALLOWED_HOSTS=1.2.3.4
BALAUR_ALLOWED_HOSTS ?= 192.168.50.12,100.124.242.131,balaur,balaur.local
export BALAUR_ALLOWED_HOSTS

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
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		go run github.com/air-verse/air@latest; \
	fi

run:
	go run . serve

seed:
	go run . seed $(ARGS)

build:
	CGO_ENABLED=0 go build -o balaur .

install-user-service: build
	mkdir -p "$(BALAUR_BIN_DIR)" "$(BALAUR_CONFIG_DIR)" "$(HOME)/.local/share/balaur/pb_data" "$(SYSTEMD_USER_DIR)"
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

lint: fmt vet test
