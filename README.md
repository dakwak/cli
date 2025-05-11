---

````markdown
# dakwak ClI 

A self-hosted tunneling CLI tool to expose your local services securely via WebSocket tunnels.

By default, the Dakwak Transformation Engine (DTE) is disabled — but it can be enabled with a simple flag when transformation features are needed (e.g., localization, rewriting, compliance).

## ✨ Features

- 🔒 Secure WebSocket connection to Dakwak Tunnel server
- 🔁 Optional integration with Dakwak Transformation Engine
- 📡 Public URLs for internal services (e.g., `https://session-*.tunnel.dakwak.com`)
- 🧠 Token + optional API key authentication
- 🐳 Docker-compatible (via `--local` flag)
- 🧵 Supports large streaming responses (up to GBs)

## 🚀 Usage

```bash
dakwak --token <TOKEN> [--apikey <APIKEY>] [--host tunnel.dakwak.com:443] [--local <internal-host>] http <host:port>
````

### Examples

Expose local app on `localhost:3000`:

```bash
dakwak --token abc123 http localhost:3000
```

Expose service in Docker named `myapp:3000`:

```bash
dakwak --token abc123 http myapp:3000
```

Use a fixed client ID (API key):

```bash
dakwak --token abc123 --apikey session-xyz http localhost:4000
```
or using DAKWAK_TOKEN env variable instead of the --token flag

## 🛠 Build Locally

```bash
go build -o dakwak main.go
```

## 🔐 Flags

* `--token`: (required) your authentication token
* `--apikey`: optional, bind to specific tunnel ID
* `--host`: override tunnel host (default: `tunnel.dakwak.com:443`)
* `--local`: use an internal host instead of IP from `<host:port>`

## 📦 Binary Output

By default, the CLI builds to `dakwak`. Release assets include binaries for:

* Linux
* macOS (darwin)
* Windows

