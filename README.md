# Open LDAP Studio

[![CI](https://github.com/OLDAPS/open-ldap-studio/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/OLDAPS/open-ldap-studio/actions/workflows/ci.yml)
[![License: Apache-2.0](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

A cross-platform desktop workbench for people who operate and debug LDAP
directories: directory administrators, identity and platform engineers, and
developers whose applications bind against a directory.

It aims to cover what practitioners use Apache Directory Studio for (connection
management, tree browsing, entry editing, raw filter search, schema work, LDIF
interchange) and to add a safer way to change a directory: secrets live in the
operating system's credential store and never in the application's own files,
and every write is previewed as an attribute-level diff before it reaches the
server.

## Status

**Early development. There is no release yet.** The first release, `v0.1.0`, is a
read-only explorer: connect, browse the tree, search with a raw RFC 4515 filter.
Editing, LDIF import and export, comparison and the rest follow after it.

[`docs/MVP.md`](docs/MVP.md) maps the milestones onto the issue tracker and says
where the MVP line falls. [`specs/`](specs/001-open-ldap-studio) holds the full
v1 specification.

## Installing

Installers are published on the
[Releases page](https://github.com/OLDAPS/open-ldap-studio/releases) once a
release exists. Each release carries a `SHA256SUMS` file.

| Platform | Download |
|---|---|
| Linux | `.deb` (needs `libgtk-3-0` and `libwebkit2gtk-4.1-0`), or a portable `.tar.gz` |
| macOS | `.dmg` (universal) |
| Windows | `-setup.exe` |

The installers are **unsigned**, so expect a warning on first launch: macOS
Gatekeeper ("Open Anyway" in System Settings, Privacy & Security) and Windows
SmartScreen ("More info", then "Run anyway"). Signing is planned.

Until then, every pull request uploads installers as workflow artifacts on its
`Package` run, if you want to try a build.

## Building from source

You need Go, Node.js 22 or later, and the
[Wails](https://wails.io/docs/gettingstarted/installation) CLI. On Linux, also
`libgtk-3-dev` and `libwebkit2gtk-4.1-dev`; on macOS the Xcode command line
tools; on Windows the WebView2 runtime. See [CONTRIBUTING.md](CONTRIBUTING.md)
for the details.

```bash
make build        # production binary in build/bin/
make run          # development mode with live reload
make test         # unit tests, Go and frontend
make lint
```

To develop against a real directory, start the seeded OpenLDAP in
[`server/`](server/README.md):

```bash
cd server && docker compose up -d
```

The integration tests (`make test-integration`) start their own directory with
testcontainers, so they need a running Docker daemon.

## How it is built

A Go core behind a [Wails v2](https://wails.io/) shell, with a React 19 and
TypeScript frontend. The Go side is a modular monolith organised by capability
under `internal/`, and the frontend never talks to Wails except through one typed
client. Every LDAP write goes through a single preview-then-commit pipeline.
[`docs/architecture.md`](docs/architecture.md) has the design.

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md). It covers the build prerequisites, the
git hooks that run CI's lint, format and unit-test checks before you push
(`make tools hooks`), and how packaging works. Commit subjects and pull request
titles follow [Conventional Commits](https://www.conventionalcommits.org/);
releases are cut from them.

## License

[Apache License 2.0](LICENSE).
