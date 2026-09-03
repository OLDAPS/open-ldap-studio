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
