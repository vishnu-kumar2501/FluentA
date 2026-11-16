# FluentA Free Cloud AI Proxy (Cloudflare Worker)

This lightweight Cloudflare Worker sits between your users' FluentA desktop apps and your AI provider (Groq or DeepSeek).

### Why use this proxy?
1. **Zero Secret Leaks:** Your private Groq / DeepSeek API key stays securely in Cloudflare and is never exposed in the desktop app binary.
2. **100% Free:** Cloudflare Workers free plan gives **100,000 requests per day** for $0/month.
3. **Global CDN Speed:** Runs at the edge in 300+ cities worldwide with minimal latency (~10-20ms overhead).

---

### Setup in 2 Minutes:

1. **Get a free Groq API Key:**
   - Sign up at [console.groq.com](https://console.groq.com)
   - Go to **API Keys** -> click **Create API Key** (e.g. `gsk_...`).

2. **Deploy on Cloudflare:**
   - Sign up / log in to [dash.cloudflare.com](https://dash.cloudflare.com)
   - In the left sidebar, click **Compute (Workers) -> Workers & Pages**
   - Click **Create Application** -> **Create Worker**
   - Give it a name (e.g., `fluenta-api`) and click **Deploy**
   - Click **Edit code**, paste the contents of `worker.js`, and click **Deploy**

3. **Add your secret key:**
   - In your Worker page, go to **Settings** -> **Variables and Secrets**
   - Click **Add** under *Secrets*:
     - **Variable name:** `AI_API_KEY`
     - **Value:** Paste your `gsk_...` key
     - Click **Save and deploy**

4. **Copy your worker URL:**
   - Your endpoint is ready: `https://fluenta-api.<your-subdomain>.workers.dev`
   - Paste this URL into FluentA Settings (or set it as the default `DefaultCloudURL`).
