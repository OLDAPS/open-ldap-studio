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

## Packaging

CI turns each build into what a user installs and installs it on the runner to
prove it starts (`.github/workflows/package.yml`, shared by pull requests and
releases). Every pull request uploads the installers as workflow artifacts, so
you can download and try one.

| Platform | Artefact | Built with |
|---|---|---|
| Linux | `.deb` and a portable `.tar.gz` | [nfpm](https://nfpm.goreleaser.com/) (`config: build/linux/nfpm.yaml`) |
| macOS | `.dmg` holding the universal `.app` | `hdiutil` |
| Windows | NSIS `-setup.exe` | `wails build -nsis` (needs `makensis`) |

All are **unsigned**, so macOS Gatekeeper and Windows SmartScreen will warn on
first launch. Signing and notarisation are tracked as T329/T330. To package
locally, run `wails build` for your platform, then
`scripts/package.sh <linux|darwin|windows> <version> <outdir>`; the version
comes from `internal/version/version.go`. A manual dry run of the whole release
packaging is `gh workflow run release.yml`, which publishes nothing.

## Spec-driven workflow

Behaviour is specified before it is built. The specification, plan and tasks are
in [`specs/`](specs/README.md), and the rules every change must satisfy are in the
[constitution](.specify/memory/constitution.md); [`docs/spec-driven-development.md`](docs/spec-driven-development.md)
explains the process.

- A fresh clone has no `.specify/feature.json` (it is per-checkout and ignored),
  so the spec-kit commands cannot find the feature. Create it with
  `printf '{"feature_directory": "specs/001-open-ldap-studio"}\n' > .specify/feature.json`
  or set `SPECIFY_FEATURE_DIRECTORY`. Details in the SDD doc.
- A change to what the product does starts as a change to the spec, not the code.
- Pick work from the issue tracker, whose issues carry the `T###` task ids.
  `specs/.../tasks.md` is the plan of record, but its checkboxes are not kept up
  to date.
- A protocol-facing change needs a test that fails first, run against a real
  directory (Principle V).
- Amend the constitution in a pull request of its own, with a version bump.

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
