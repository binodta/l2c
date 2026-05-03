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

      console.log("WebSocket accepted");
      return new Response(null, { status: 101, webSocket: client });
    }

    // Handle public traffic to be proxied
    if (!this.clientWs) {
      return new Response(landingPageHTML, { 
        status: 503,
        headers: { "Content-Type": "text/html; charset=utf-8" }
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

    const proxyRequest = {
      type: "req",
      id: requestId,
      method: request.method,
      url: url.pathname + url.search,
      headers: Object.fromEntries(request.headers),
      body: bodyBase64,
    };

    return new Promise((resolve) => {
      const timeout = setTimeout(() => {
        this.pendingRequests.delete(requestId);
        resolve(new Response("Gateway Timeout", { status: 504 }));
      }, 10000);

      this.pendingRequests.set(requestId, (response: any) => {
        clearTimeout(timeout);
        const headers = new Headers(response.headers);
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
    }
  }

  async webSocketError(ws: WebSocket, error: any) {
    if (this.clientWs === ws) {
      this.clientWs = null;
    }
  }
}
