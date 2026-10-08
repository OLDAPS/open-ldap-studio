.PHONY: build test test-functional test-integration lint fmt run gates

build:
	wails build -tags webkit2_41

test:
	go test -short ./...
	cd frontend && npm run test

# The live tests in internal/bridge run against the seeded OpenLDAP fixture in
# server/. REQUIRE_LIVE_DIRECTORY turns a missing fixture into a failure rather
# than a skip; the fixture is torn down afterwards whether or not tests pass.
test-functional:
	cd server && docker compose up -d
	@echo "waiting for the directory to answer a search..."
	@for i in $$(seq 1 60); do \
		[ "$$(docker inspect -f '{{.State.Health.Status}}' ldap 2>/dev/null)" = healthy ] && exit 0; \
		sleep 3; \
	done; echo "directory never became healthy"; docker logs ldap | tail -30; cd server && docker compose down -v; exit 1
	REQUIRE_LIVE_DIRECTORY=1 go test -count=1 -run Live ./internal/bridge/; rc=$$?; \
		cd server && docker compose down -v; exit $$rc

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
