/**
 * FluentA Cloudflare Worker Proxy with Cloudflare Native Rate Limiting
 */
export default {
  async fetch(request, env) {
    // 1. Handle CORS preflight
    if (request.method === "OPTIONS") {
      return new Response(null, {
        headers: {
          "Access-Control-Allow-Origin": "*",
          "Access-Control-Allow-Methods": "GET, POST, OPTIONS",
          "Access-Control-Allow-Headers": "Content-Type, Authorization",
        },
      });
    }

    const url = new URL(request.url);

    // 2. Health check endpoint (exempt from rate limits)
    if (url.pathname === "/models" || url.pathname === "/v1/models" || url.pathname === "/health") {
      return new Response(JSON.stringify({ status: "ok", service: "FluentA Cloud AI Gateway" }), {
        status: 200,
        headers: { "Content-Type": "application/json", "Access-Control-Allow-Origin": "*" }
      });
    }

    // 3. Enforce Cloudflare Native Rate Limiter (30 requests/min per IP)
    if (env.RATE_LIMITER) {
      const clientIP = request.headers.get("cf-connecting-ip") || "unknown";
      const { success } = await env.RATE_LIMITER.limit({ key: clientIP });
      if (!success) {
        return new Response(JSON.stringify({
          error: { message: "Too many checks. Please wait 1 minute before trying again.", type: "rate_limit_error" }
        }), {
          status: 429,
          headers: {
            "Content-Type": "application/json",
            "Access-Control-Allow-Origin": "*",
            "Retry-After": "60"
          }
        });
      }
    }

    // 4. Verify Secret Key exists
    const apiKey = env.AI_API_KEY;
    if (!apiKey) {
      return new Response(JSON.stringify({ error: "Gateway configuration error: missing AI_API_KEY secret" }), {
        status: 500,
        headers: { "Content-Type": "application/json", "Access-Control-Allow-Origin": "*" }
      });
    }

    // 5. Proxy chat completions to Groq
    const targetBase = (env.AI_BASE_URL || "https://api.groq.com/openai/v1").replace(/\/+$/, "");
    const targetURL = `${targetBase}/chat/completions`;
    const newHeaders = new Headers(request.headers);
    newHeaders.set("Authorization", `Bearer ${apiKey}`);
    newHeaders.set("User-Agent", "FluentA-Proxy/1.0");

    try {
      const response = await fetch(targetURL, {
        method: request.method,
        headers: newHeaders,
        body: request.body,
      });

      const responseHeaders = new Headers(response.headers);
      responseHeaders.set("Access-Control-Allow-Origin", "*");
      return new Response(response.body, {
        status: response.status,
        statusText: response.statusText,
        headers: responseHeaders,
      });
    } catch (err) {
      return new Response(JSON.stringify({ error: `Gateway error: ${err.message}` }), {
        status: 502,
        headers: { "Content-Type": "application/json", "Access-Control-Allow-Origin": "*" }
      });
    }
  }
};