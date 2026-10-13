import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  /* config options here */
  output: "standalone",
  async rewrites() {
    return [
      {
        source: '/backend-api/:path*',
        // Reads the runtime environment variable injected by your Kubernetes ConfigMap
        destination: `${process.env.BACKEND_API_URL || 'http://localhost:8080'}/:path*`,
      },
    ]
  },
};

export default nextConfig;
