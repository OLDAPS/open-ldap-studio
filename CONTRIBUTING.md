# Contributing to Open LDAP Studio

## Prerequisites

To build and run this project, you will need:
- Go 1.26 or later
- Node.js 22 or later
- Wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

### Platform-specific requirements

#### Linux
```bash
sudo apt-get install libwebkit2gtk-4.1-dev
```

#### macOS
Xcode Command Line Tools:
```bash
xcode-select --install
```

#### Windows
WebView2 Runtime (usually pre-installed on Windows 11).

## Building
```bash
make build
```

## Testing
```bash
make test
```

The integration tests (`make test-integration`) start a seeded OpenLDAP with
testcontainers, so they need a running Docker daemon and fail without one.

## Git hooks

The repository ships hooks that run the same checks as CI on your machine, so a
lint, format or unit-test failure is caught before it reaches GitHub.

```bash
make tools   # golangci-lint v2 (the version CI uses), gofumpt, govulncheck
make hooks   # git config core.hooksPath .githooks
```

| Hook | Runs | Takes |
|---|---|---|
| `pre-commit` | `gofmt` on staged Go files, prettier and eslint on staged frontend files, refuses generated or secret-bearing paths (`.env`, `frontend/dist/`, `build/bin/`) | seconds |
| `commit-msg` | the subject must be a [conventional commit](https://www.conventionalcommits.org/) (`feat`, `fix`, `perf`, `revert`, `refactor`, `docs`, `build`, `ci`, `test`, `style`, `chore`) | instant |
| `pre-push` | for the files the push changes: `go vet`, `golangci-lint`, `go test -short`; frontend lint, prettier and tests. A side with no changes is skipped | under a minute |

The first push builds `frontend/dist` once, because `main.go` embeds it and
nothing in the Go module compiles without it.

Not run locally: the Docker integration tests, the three-OS build, `govulncheck`
and `npm audit` (`make gates` runs the last two). CI stays the authority; the
hooks only save the round trip. To skip them once, use `git commit --no-verify`
or `git push --no-verify`; to turn them off, `make unhooks`.
