# Prod data dir — the real personal data `make run` serves. Override if you keep
# it elsewhere: make run BALAUR_DATA_DIR=/path/to/pb_data
BALAUR_DATA_DIR ?= $(HOME)/.local/share/balaur/pb_data
# Prod bind. 8080 so prod (run) and dev (8090) coexist; 0.0.0.0 so it's reachable
# over the NetBird mesh (ufw + the NetBird ACL gate access; BALAUR_ALLOWED_HOSTS,
# exported below, lets the host guard accept mesh hostnames).
PROD_HTTP ?= 0.0.0.0:8080

# The Go linker writes large temp objects; the host's default /tmp is a small
# tmpfs and the linker OOMs there (`signal: killed`) during test/lint/vulncheck.
# Point TMPDIR at real disk for every child `go` process. Override via env if
# your /tmp is real disk: make test TMPDIR=/tmp
TMPDIR ?= $(HOME)/.cache/go-tmp
export TMPDIR
$(shell mkdir -p $(TMPDIR))

.PHONY: help tools dev dev-port-open dev-port-close run seed build test race fmt vet staticcheck vulncheck lint

help:
	@echo "make run    # serve PROD: real data ($(BALAUR_DATA_DIR)), port 8080 — run inside a zellij session to persist"
	@echo "make dev    # hot-reload DEV via air: throwaway repo pb_data/, port $(DEV_PORT); opens :$(DEV_PORT) on $(NETBIRD_IFACE) while running, closes on exit"
	@echo "make tools  # install dlv + air, activate the pre-commit lint hook"
	@echo "make dev-port-open   # open dev :$(DEV_PORT) over NetBird by hand (sudo ufw)"
	@echo "make dev-port-close  # close dev :$(DEV_PORT) again (sudo ufw)"
	@echo "make build  # build a CGO-free binary"
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

# Serve the prod instance (real data dir, mesh-reachable). No daemon: run this in
# a long-lived zellij session so it survives SSH logout. `make dev` (port 8090,
# repo-local pb_data) runs alongside it for hot-reload development.
run:
	go run . serve --http $(PROD_HTTP) --dir $(BALAUR_DATA_DIR)

seed:
	go run . seed $(ARGS)

build:
	CGO_ENABLED=0 go build -o balaur .

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
	go tool staticcheck ./...

vulncheck:
	go tool govulncheck ./...

lint: fmt vet staticcheck test
