.PHONY: build test test-integration lint fmt run gates

build:
	wails build -tags webkit2_41

test:
	go test -short ./...
	cd frontend && npm run test

test-integration:
	go test -tags=integration ./...

lint:
	golangci-lint run
	cd frontend && npm run lint

fmt:
	gofumpt -l -w .
	cd frontend && npm run format

run:
	wails dev -tags webkit2_41

gates: lint test
	govulncheck ./...
	cd frontend && npm audit --audit-level=high
	go mod verify
