import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  reactStrictMode: true,
  output: "standalone",
  allowedDevOrigins: ["127.0.0.1", "localhost"],
  outputFileTracingIncludes: { "/lesson-content/*": ["./lesson-content/*.html"] },
  async headers() {
    const origin = new URL(process.env.HTML_PREVIEW_PUBLIC_URL ?? "http://html.localhost:8081").origin;
    return ["/tasks/library/html/:path*", "/tasks/:assignmentId"].map(source => ({ source, headers: [{ key: "Content-Security-Policy", value: `frame-src ${origin}; object-src 'none'; base-uri 'self'` }] }));
  },
  async rewrites() {
    return [{ source: "/api/v1/:path*", destination: `${process.env.API_INTERNAL_URL ?? "http://127.0.0.1:8080"}/api/v1/:path*` }];
  }
};

export default nextConfig;
