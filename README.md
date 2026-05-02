# l2c-proxy (Local to Cloud Proxy)

A simple, ngrok-like tunnel built with Cloudflare Workers (Durable Objects) and Golang. Optimized for the Cloudflare Free Tier.

## Features
- **Multi-App Support**: Map multiple local services in a single config file.
- **Concurrent Requests**: Handles multiple simultaneous requests over a single WebSocket.
- **Path-based Routing**: No custom domain required (uses `*.workers.dev`).
- **Cross-Platform**: Works on Linux, macOS, and Windows.

## Prerequisites
- **Go 1.20+**
- **Node.js & pnpm** (for the worker)
- **Make** (optional, but recommended for Linux/macOS)

## Getting Started

### 1. Deploy the Cloudflare Worker

```bash
cd worker
pnpm install
npx wrangler deploy
```

### 2. Configure the Client

Copy the example config and edit it with your tunnel IDs and local ports:

```bash
cp config.json.example config.json
```

**config.json example:**
```json
{
  "server": "l2c-proxy.bomzzn.workers.dev",
  "tunnels": [
    { "id": "my-app", "local": "http://localhost:8000" },
    { "id": "api",    "local": "http://localhost:8080" }
  ]
}
```

### 3. Run the Client

```bash
make run
```

Your services will be available at:
`https://l2c-proxy.bomzzn.workers.dev/t/{id}/`

## Testing Locally

1. Start the test server: `make test-server` (runs on port 8000).
2. Ensure `config.json` has a tunnel for `localhost:8000`.
3. Run the client: `make run`.
4. Visit the URL provided in the terminal. You should see a **"It works!"** message.

## OS Specific Instructions

### Linux & macOS
The easiest way is to use the provided `Makefile`:
```bash
make test-server  # Terminal 1
make run          # Terminal 2
```

### Windows
#### Option 1: WSL (Recommended)
Follow the Linux instructions inside your WSL terminal.

#### Option 2: PowerShell
If you don't have `make` installed, run the commands manually:
```powershell
# Terminal 1: Test Server
go run cli/test-server/main.go

# Terminal 2: Tunnel Client
go run cli/main.go -config config.json
```

## How it works
1. The **Go Client** reads `config.json` and establishes WebSocket connections for each tunnel.
2. The Worker uses **Durable Objects** to maintain these connections.
3. Requests to `/t/{id}/*` are pushed over the WebSocket to your local machine.
4. The Go Client proxies the request to your local port and returns the response.
