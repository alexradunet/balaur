.PHONY: help dev run build test fmt vet lint

help:
	@echo "make dev    # start Balaur with hot reload via air"
	@echo "make run    # start Balaur once (go run . serve)"
	@echo "make build  # build a CGO-free binary"
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

test:
	go test ./...

fmt:
	@[ "$(shell gofmt -l .)" = "" ]

vet:
	go vet ./...

lint: fmt vet test
