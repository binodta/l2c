# l2c — Local to Cloud

> A lightweight, ngrok-like tunneling tool powered by Cloudflare Workers. Expose any local service to the internet instantly — no open ports, no firewalls, no paid plans.

[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-blue)](https://github.com/binodta/l2c)
[![Powered by](https://img.shields.io/badge/powered%20by-Cloudflare%20Workers-orange)](https://workers.cloudflare.com/)

---

## Features

- 🔒 **Secure** — Token-based authentication baked into every request
- 🔄 **Auto-Reconnect** — Resilient WebSocket with exponential backoff
- 🗺️ **Multi-Tunnel** — Expose multiple local services simultaneously
- 🌐 **Custom Domains** — Bring your own domain natively
- ⚡ **Zero Infrastructure** — Runs entirely on Cloudflare's free tier
- 📦 **No Dependencies** — Single binary, no Go/Node required to run
- 🌍 **Cross-Platform** — Linux, macOS (Intel & Apple Silicon), Windows (WSL & native)

---

## Installation

### Linux & macOS (one-liner)

```bash
curl -fsSL https://raw.githubusercontent.com/binodta/l2c/main/scripts/install.sh | bash
```

### Windows — WSL (Recommended)

Open a WSL terminal and run the same command:

```bash
curl -fsSL https://raw.githubusercontent.com/binodta/l2c/main/scripts/install.sh | bash
```

> **Don't have WSL?** Install it with: `wsl --install` in PowerShell, then restart.

### Windows — Native PowerShell

```powershell
irm https://raw.githubusercontent.com/binodta/l2c/main/scripts/install.ps1 | iex
```

The installer will:
1. Auto-detect your OS and architecture
2. Download the correct pre-built binary (~5MB)
3. Add `l2c` to your `PATH` automatically
4. Launch the interactive setup wizard

---

## Quick Start

After installation, activate `l2c` in your current terminal:

```bash
export PATH="$PATH:$HOME/.l2c"   # Linux/macOS/WSL
```

Then run the interactive setup (deploys your Cloudflare Worker):

```bash
l2c setup
```

Start tunneling:

```bash
l2c run
```

Your local services are now live at:
```
https://<your-worker>.workers.dev/<tunnel-id>/
```
*(Or your custom domain if configured)*

---

## Prerequisites

- A **Cloudflare account** (free tier works)
- **Node.js** — required only during `l2c setup` to deploy the worker via `npx wrangler`
- *(Optional)* **Cloudflare Domain** — to use the custom domain feature, the domain must be active as a Zone in your Cloudflare account.

> After setup, only the `l2c` binary is needed to run tunnels.

---

## Configuration

The config is saved automatically at `~/.l2c/config.json` during setup.

```json
{
  "worker_url": "your-worker.workers.dev",
  "custom_domain": "api.example.com",
  "token": "your-secret-token",
  "tunnels": [
    { "id": "app",  "local": "http://localhost:3000" },
    { "id": "api",  "local": "http://localhost:8080" }
  ]
}
```

You can edit this file manually, or use the built-in CLI commands to manage your tunnels:

```bash
l2c tunnel add --id new-app --local http://localhost:4000
l2c tunnel list
l2c tunnel remove --id old-app
```

After making changes, simply restart `l2c run` for them to take effect.

---

## Commands

| Command | Description |
|---|---|
| `l2c setup` | Deploy worker & configure credentials interactively |
| `l2c domain <domain>` | Set or update your custom domain |
| `l2c run` | Start all tunnels defined in config |
| `l2c run --config /path/to/config.json` | Use a custom config file |
| `l2c tunnel add --id <id> --local <url>` | Add a new tunnel |
| `l2c tunnel list` | List all configured tunnels |
| `l2c tunnel remove --id <id>` | Remove a tunnel (use `--force` to skip prompt) |

---

## How It Works

```
Internet Request
      │
      ▼
Cloudflare Worker  ──── WebSocket ────  l2c (your machine)
  (*.workers.dev)                            │
                                             ▼
                                      localhost:PORT
```

1. **`l2c run`** connects to your Cloudflare Worker over a persistent WebSocket.
2. Incoming requests to `https://<worker>/<id>/` are forwarded over the socket.
3. `l2c` proxies the request to your local service and returns the response.
4. If the connection drops, `l2c` automatically reconnects with exponential backoff.

---

## Error Reference

| Error | Meaning |
|---|---|
| `Authentication failed (HTTP 401)` | Wrong token — check `~/.l2c/config.json` |
| `Tunnel endpoint not found (HTTP 404)` | Worker not deployed — run `l2c setup` |
| `Cannot reach worker` | Check internet / worker URL |
| `Local server not reachable` | Your local app isn't running on the configured port |

---

## License

MIT
