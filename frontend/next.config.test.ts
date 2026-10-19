import { afterEach, describe, expect, it } from "vitest";

import nextConfig from "./next.config";

describe("nextConfig rewrites", () => {
  afterEach(() => {
    delete process.env.BACKEND_API_URL;
  });

  it("proxies /backend-api to localhost by default", async () => {
    delete process.env.BACKEND_API_URL;
    const rewrites = await nextConfig.rewrites?.();

    expect(rewrites).toEqual([
      {
        source: "/backend-api/:path*",
        destination: "http://localhost:8080/:path*",
      },
    ]);
  });

  it("uses BACKEND_API_URL when set", async () => {
    process.env.BACKEND_API_URL =
      "http://kubeai-backend.kubeai-training.svc.cluster.local:8080";
    const rewrites = await nextConfig.rewrites?.();

    expect(rewrites).toEqual([
      {
        source: "/backend-api/:path*",
        destination:
          "http://kubeai-backend.kubeai-training.svc.cluster.local:8080/:path*",
      },
    ]);
  });
});
