# Keep in step with the golangci-lint version in .github/workflows/ci.yml;
# .githooks/tests/run.sh fails when the two differ.
GOLANGCI_LINT_VERSION := v2.14.0
WAILS_FLAGS := $(if $(filter linux,$(shell go env GOOS)),-tags webkit2_41,)

.PHONY: build bindings test test-integration lint fmt run gates tools hooks unhooks test-hooks spec-check test-spec

build:
	wails build $(WAILS_FLAGS)

# Generated from the objects bound in main.go; never edit wailsjs by hand.
bindings:
	wails generate module

test:
	go test -short ./...
	cd frontend && npm run test

# Needs a running Docker daemon: the live tests in internal/bridge start a
# seeded OpenLDAP with testcontainers, and fail rather than skip without one.
test-integration:
	go test -tags=integration ./...

lint:
	golangci-lint run
	cd frontend && npm run lint

fmt:
	gofumpt -l -w .
	cd frontend && npm run format

run:
	wails dev $(WAILS_FLAGS)

gates: lint test
	govulncheck ./...
	cd frontend && npm audit --audit-level=high
	go mod verify

# The tool versions CI uses, installed into $(go env GOPATH)/bin.
tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	go install mvdan.cc/gofumpt@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest

# Turn the repository's git hooks on (and off). See CONTRIBUTING.md.
hooks:
	git config core.hooksPath .githooks
	chmod +x .githooks/commit-msg .githooks/pre-commit .githooks/pre-push .githooks/tests/run.sh
	@echo "git hooks enabled: pre-commit, commit-msg, pre-push (skip once with --no-verify)"

unhooks:
	git config --unset core.hooksPath

test-hooks:
	bash .githooks/tests/run.sh

# Spec tooling (scripts/spec, Python standard library only; run as `python3 file`).
spec-check:
	python3 scripts/spec/check.py --notes

test-spec:
	python3 -m unittest discover -s scripts/spec/tests
