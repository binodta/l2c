import { landingPageHTML } from "./html";

export class Tunnel implements DurableObject {
  private state: DurableObjectState;
  private clientWs: WebSocket | null = null;
  private pendingRequests: Map<string, (res: any) => void> = new Map();

  constructor(state: DurableObjectState) {
    this.state = state;
    
    // Recover WebSockets if the DO was hibernated
    this.state.getWebSockets().forEach(ws => {
      this.clientWs = ws;
    });
  }

  async fetch(request: Request): Promise<Response> {
    const url = new URL(request.url);
    console.log(`DO fetch: ${url.pathname}`);

    // Internal cleanup request
    if (url.pathname === "/internal-cleanup") {
      if (!this.clientWs && this.state.getWebSockets().length === 0) {
        console.log("Internal cleanup: No connections, deleting storage");
        await this.state.storage.deleteAll();
        return new Response("Cleaned up", { status: 200 });
      }
      return new Response("In use", { status: 409 });
    }

    // Handle WebSocket connection from local client
    if (url.pathname.startsWith("/connect")) {
      console.log("Handling /connect WebSocket upgrade");
      
      // Check if a client is already connected
      if (this.clientWs || this.state.getWebSockets().length > 0) {
        console.log("Rejecting connection: Tunnel ID already in use");
        return new Response("Tunnel ID already in use", { status: 409 });
      }

      if (request.headers.get("Upgrade") !== "websocket") {
        return new Response("Expected Upgrade: websocket", { status: 426 });
      }

      const pair = new WebSocketPair();
      const [client, server] = Object.values(pair);

      this.state.acceptWebSocket(server);
      this.clientWs = server;

      // Clear any pending cleanup alarm
      await this.state.storage.deleteAlarm();

      console.log("WebSocket accepted");
      return new Response(null, { status: 101, webSocket: client });
    }

    // Handle public traffic to be proxied
    if (!this.clientWs) {
      return new Response(landingPageHTML, { 
        status: 503,
        headers: { 
          "Content-Type": "text/html; charset=utf-8",
          "X-L2C-Disconnected": "true"
        }
      });
    }

    const requestId = crypto.randomUUID();
    const bodyBuffer = await request.arrayBuffer();
    
    // Convert ArrayBuffer to Base64 safely for any data type
    let bodyBase64: string | null = null;
    if (bodyBuffer.byteLength > 0) {
      const bytes = new Uint8Array(bodyBuffer);
      let binary = "";
      for (let i = 0; i < bytes.byteLength; i++) {
        binary += String.fromCharCode(bytes[i]);
      }
      bodyBase64 = btoa(binary);
    }

    const headers: Record<string, string[]> = {};
    for (const [key, value] of request.headers.entries()) {
      headers[key] = [value];
    }

    const proxyRequest = {
      type: "req",
      id: requestId,
      method: request.method,
      url: url.pathname + url.search,
      headers: headers,
      body: bodyBase64,
    };

    return new Promise((resolve) => {
      const timeout = setTimeout(() => {
        this.pendingRequests.delete(requestId);
        resolve(new Response("Gateway Timeout", { status: 504 }));
      }, 10000);

      this.pendingRequests.set(requestId, (response: any) => {
        clearTimeout(timeout);
        const headers = new Headers();
        if (response.headers) {
          for (const [k, values] of Object.entries(response.headers)) {
            for (const v of (values as string[])) {
              headers.append(k, v);
            }
          }
        }
        const bodyBuffer = response.body ? Uint8Array.from(atob(response.body), c => c.charCodeAt(0)) : null;
        resolve(new Response(bodyBuffer, {
          status: response.status,
          headers: headers
        }));
      });

      this.clientWs!.send(JSON.stringify(proxyRequest));
    });
  }

  async webSocketMessage(ws: WebSocket, message: string | ArrayBuffer) {
    if (typeof message !== "string") return;

    try {
      const data = JSON.parse(message);
      if (data.type === "res") {
        const callback = this.pendingRequests.get(data.id);
        if (callback) {
          callback(data);
          this.pendingRequests.delete(data.id);
        }
      }
    } catch (e) {
      console.error("Error parsing WebSocket message:", e);
    }
  }

  async webSocketClose(ws: WebSocket, code: number, reason: string, wasClean: boolean) {
    if (this.clientWs === ws) {
      this.clientWs = null;
      // Schedule cleanup in 10 minutes if no other connections
      if (this.state.getWebSockets().length === 0) {
        await this.state.storage.setAlarm(Date.now() + 10 * 60 * 1000);
      }
    }
  }

  async webSocketError(ws: WebSocket, error: any) {
    if (this.clientWs === ws) {
      this.clientWs = null;
      // Schedule cleanup in 10 minutes if no other connections
      if (this.state.getWebSockets().length === 0) {
        await this.state.storage.setAlarm(Date.now() + 10 * 60 * 1000);
      }
    }
  }

  async alarm() {
    console.log("Cleanup alarm fired");
    if (!this.clientWs && this.state.getWebSockets().length === 0) {
      console.log("No connections found, deleting storage");
      await this.state.storage.deleteAll();
    } else {
      console.log("Connections found, skipping cleanup");
    }
  }
}
