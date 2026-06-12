PREFIX ?= $(HOME)/.local
BALAUR_BIN_DIR ?= $(PREFIX)/bin
BALAUR_CONFIG_DIR ?= $(HOME)/.config/balaur
SYSTEMD_USER_DIR ?= $(HOME)/.config/systemd/user
SERVICE_NAME ?= balaur

.PHONY: help dev run build install-user-service start-user-service stop-user-service restart-user-service status-user-service logs-user-service test fmt vet lint

help:
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
	@echo "make fmt    # gofmt -l ."
	@echo "make vet    # go vet ./..."
	@echo "make lint   # fmt + vet + test"

dev:
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		go run github.com/air-verse/air@latest; \
	fi

run:
	go run . serve

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

fmt:
	@[ "$(shell gofmt -l .)" = "" ]

vet:
	go vet ./...

lint: fmt vet test
