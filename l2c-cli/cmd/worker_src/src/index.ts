import { Tunnel } from "./tunnel";
import { landingPageHTML, getErrorPage } from "./html";

export interface Env {
  TUNNELS: DurableObjectNamespace;
  AUTH_TOKEN?: string;
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const url = new URL(request.url);

    // Client registration: /connect/:id
    const connectMatch = url.pathname.match(/^\/connect\/([^\/]+)/);
    if (connectMatch) {
      // Authentication check for CLI connection
      const authToken = env.AUTH_TOKEN;
      if (authToken) {
        const authHeader = request.headers.get("Authorization");
        const urlToken = url.searchParams.get("token");
        
        if (authHeader !== `Bearer ${authToken}` && urlToken !== authToken) {
          return new Response("Unauthorized: Provide valid token to connect tunnel", { status: 401 });
        }
      }

      const tunnelId = connectMatch[1];
      console.log(`Connecting tunnel: ${tunnelId}`);
      const id = env.TUNNELS.idFromName(tunnelId);
      const obj = env.TUNNELS.get(id);
      return obj.fetch(request);
    }

    // Public traffic: /:id/*
    const publicMatch = url.pathname.match(/^\/([^\/]+)(\/.*)?/);
    if (publicMatch) {
      let tunnelId = publicMatch[1];
      let remainingPath = publicMatch[2] || "/";
      
      const doFetch = async (tid: string, path: string) => {
        const id = env.TUNNELS.idFromName(tid);
        const obj = env.TUNNELS.get(id);
        const newUrl = new URL(url);
        newUrl.pathname = path;
        const newRequest = new Request(newUrl.toString(), request);
        return obj.fetch(newRequest);
      };

      let response = await doFetch(tunnelId, remainingPath);
      let activeTunnelId = tunnelId;

      // SPA / React App Fallback Routing
      // If the parsed tunnel is not connected (503 with X-L2C-Disconnected), 
      // it might be an absolute path asset request like /assets/main.js
      if (response.status === 503 && response.headers.get("X-L2C-Disconnected") === "true") {
        let fallbackTunnelId: string | null = null;
        
        // 1. Try Referer Header (e.g. referer: https://server/app-one/index.html)
        const referer = request.headers.get("Referer");
        if (referer) {
          try {
            const refererUrl = new URL(referer);
            if (refererUrl.host === url.host) {
              const refMatch = refererUrl.pathname.match(/^\/([^\/]+)/);
              if (refMatch) fallbackTunnelId = refMatch[1];
            }
          } catch (e) {}
        }
        
        // 2. Try Cookie Fallback
        if (!fallbackTunnelId) {
          const cookieHeader = request.headers.get("Cookie") || "";
          const cookieMatch = cookieHeader.match(/l2c-active-tunnel=([^;]+)/);
          if (cookieMatch) {
            const tid = cookieMatch[1];
            const isNavigate = request.headers.get("Sec-Fetch-Mode") === "navigate";
            const hasExtension = url.pathname.includes(".");
            
            // Only fallback via cookie for navigations if they look like assets (have extension)
            // This prevents /app-two falling back to app-one just because of a cookie.
            // Referer-based fallback still works for all types.
            if (!isNavigate || hasExtension) {
              fallbackTunnelId = tid;
            }
          }
        }

        if (fallbackTunnelId && fallbackTunnelId !== tunnelId) {
           // Pass the ENTIRE original pathname to the fallback tunnel
           const fallbackResponse = await doFetch(fallbackTunnelId, url.pathname);
           if (fallbackResponse.status !== 503 || fallbackResponse.headers.get("X-L2C-Disconnected") !== "true") {
             response = fallbackResponse;
             activeTunnelId = fallbackTunnelId;
           }
        }

        // Final check: if still disconnected, show a helpful error page
        if (response.status === 503 && response.headers.get("X-L2C-Disconnected") === "true") {
          return new Response(getErrorPage("Not Connected", `Tunnel <code>${tunnelId}</code> is not currently connected. Ensure your local service is running and you have started the tunnel with <code>l2c run</code>.`), {
            status: 503,
            headers: { "Content-Type": "text/html; charset=utf-8" }
          });
        }
      }

      // Modify response to inject the tracking cookie for subsequent disconnected requests
      const finalResponse = new Response(response.body, response);
      if (finalResponse.status !== 503 && finalResponse.status !== 404) {
        finalResponse.headers.append("Set-Cookie", `l2c-active-tunnel=${activeTunnelId}; Path=/; HttpOnly; SameSite=Lax`);
      }
      return finalResponse;
    }

    return new Response(landingPageHTML, { 
      status: 404,
      headers: { "Content-Type": "text/html; charset=utf-8" }
    });
  }
};

export { Tunnel };
