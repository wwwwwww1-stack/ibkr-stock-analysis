.PHONY: test test-go test-frontend build dev

WAILS ?= $(shell go env GOPATH)/bin/wails

test: test-go test-frontend

test-go:
	go test ./...

test-frontend:
	npm --prefix frontend test -- --run
	npm --prefix frontend run typecheck
	npm --prefix frontend run build

build:
	$(WAILS) build

dev:
	$(WAILS) dev
