.PHONY: test test-go test-agent test-frontend build dev

WAILS ?= $(shell go env GOPATH)/bin/wails

test: test-go test-agent test-frontend

test-go:
	go test ./...

test-agent:
	npm --prefix agent-worker test -- --run
	npm --prefix agent-worker run build

test-frontend:
	npm --prefix frontend test -- --run
	npm --prefix frontend run typecheck
	npm --prefix frontend run build

build:
	npm --prefix agent-worker run build
	$(WAILS) build

dev:
	npm --prefix agent-worker run build
	$(WAILS) dev

