# DNS Changer

A cross-platform desktop app to quickly switch between DNS profiles on macOS, Windows, and Linux. Built with [Wails](https://wails.io) v2 + Go + Svelte.

## Features

- Add, edit, delete DNS profiles (IPv4)
- Select a profile and toggle DNS on/off with a single switch
- Ping DNS servers to check latency
- Resets to DHCP when toggled off
- Persistent profile storage

## Build Commands

### Prerequisites

- [Go](https://go.dev) 1.21+
- [Wails CLI](https://wails.io/docs/gettingstarted/installation) v2: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- Node.js 18+ (for frontend dependencies)

### macOS

```bash
# Apple Silicon
wails build

# Intel
wails build -platform darwin/amd64

# Universal binary (both architectures)
wails build -platform darwin/universal
```

Output: `build/bin/Dns Changer.app`

### Windows (cross-compile from macOS/Linux)

Requires MinGW cross-compiler:

```bash
# Install toolchain (macOS)
brew install mingw-w64

# Build
wails build -platform windows/amd64

# 32-bit
wails build -platform windows/386
```

Output: `build/bin/Dns Changer.exe`

### Linux (cross-compile from macOS)

```bash
# Install musl-cross (macOS)
brew install FiloSottile/musl-cross/musl-cross

# Build
wails build -platform linux/amd64

# ARM
wails build -platform linux/arm64
```

Output: `build/bin/dns-changer`

> **Tip:** For the cleanest Linux build, compile natively on a Linux machine — no cross-compiler needed.

### Development Mode

```bash
wails dev
```

Opens the app with live-reload and a browser dev server at `http://localhost:34115`.

## Changing the App Icon

Replace `build/appicon.png` (1024×1024 PNG) and rebuild:

```bash
wails build
```

## Changing the App Name

Edit `wails.json`:

```json
"name": "your-app-id",
"outputfilename": "Your App Name"
```

Then rebuild.

## Project Structure

```
.
├── main.go              # Entry point, embeds frontend
├── app.go               # Wails bound methods
├── wails.json           # Wails configuration
├── dns/                 # Platform-specific DNS logic
│   ├── dns.go           # Manager interface
│   ├── dns_darwin.go    # macOS (networksetup)
│   ├── dns_linux.go     # Linux (resolvectl/nmcli)
│   └── dns_windows.go   # Windows (netsh)
├── profiles/            # JSON profile storage
├── pinger/              # DNS ping via TCP/UDP port 53
└── frontend/            # Svelte + Vite UI
```

---

Made by [@AMIRALITVK](https://github.com/AMIRALITVK)
