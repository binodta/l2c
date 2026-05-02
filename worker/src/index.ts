import { Tunnel } from "./tunnel";

export interface Env {
  TUNNELS: DurableObjectNamespace;
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const url = new URL(request.url);

    // Client registration: /connect/:id
    const connectMatch = url.pathname.match(/^\/connect\/([^\/]+)/);
    if (connectMatch) {
      const tunnelId = connectMatch[1];
      console.log(`Connecting tunnel: ${tunnelId}`);
      const id = env.TUNNELS.idFromName(tunnelId);
      const obj = env.TUNNELS.get(id);
      return obj.fetch(request);
    }

    // Public traffic: /t/:id/*
    const publicMatch = url.pathname.match(/^\/t\/([^\/]+)(\/.*)?/);
    if (publicMatch) {
      const tunnelId = publicMatch[1];
      const remainingPath = publicMatch[2] || "/";
      console.log(`Public request for tunnel: ${tunnelId}, path: ${remainingPath}`);
      
      const id = env.TUNNELS.idFromName(tunnelId);
      const obj = env.TUNNELS.get(id);

      // Rewrite URL to strip the /t/:id prefix before sending to DO
      const newUrl = new URL(url);
      newUrl.pathname = remainingPath;
      const newRequest = new Request(newUrl.toString(), request);

      return obj.fetch(newRequest);
    }

    return new Response("l2c-proxy: use /t/:id/ to access a tunnel", { status: 404 });
  }
};

export { Tunnel };
