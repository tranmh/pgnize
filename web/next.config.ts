import type { NextConfig } from "next";

// In dev (and any standalone deploy without an upstream proxy), Next.js rewrites
// `/api/*` and `/healthz` to the Go backend. In production behind Caddy the proxy
// usually fronts both, but these rewrites keep the app self-sufficient.
const apiBase = process.env.PGNIZE_API_URL || "http://localhost:8080";

const nextConfig: NextConfig = {
  output: "standalone",
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: `${apiBase}/api/:path*`,
      },
      {
        source: "/healthz",
        destination: `${apiBase}/healthz`,
      },
    ];
  },
};

export default nextConfig;
