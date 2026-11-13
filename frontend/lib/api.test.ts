import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

describe("fetchFromBackend", () => {
  const originalBackendUrl = process.env.BACKEND_API_URL;

  beforeEach(() => {
    vi.resetModules();
  });

  afterEach(() => {
    if (originalBackendUrl === undefined) {
      delete process.env.BACKEND_API_URL;
    } else {
      process.env.BACKEND_API_URL = originalBackendUrl;
    }
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("calls the configured backend URL and parses JSON responses", async () => {
    process.env.BACKEND_API_URL = "https://backend.example.test";
    const fetchSpy = vi.fn().mockResolvedValue({
      ok: true,
      json: vi.fn().mockResolvedValue({ status: "ok" }),
      text: vi.fn(),
    });
    vi.stubGlobal("fetch", fetchSpy);

    const { fetchFromBackend } = await import("./api");

    await expect(fetchFromBackend("jobs/42")).resolves.toEqual({ status: "ok" });
    expect(fetchSpy).toHaveBeenCalledWith(
      "https://backend.example.test/jobs/42",
      expect.objectContaining({
        headers: expect.objectContaining({
          "Content-Type": "application/json",
        }),
      }),
    );
  });

  it("returns text when parseAs is set to text", async () => {
    process.env.BACKEND_API_URL = "https://backend.example.test";
    const fetchSpy = vi.fn().mockResolvedValue({
      ok: true,
      text: vi.fn().mockResolvedValue("job log output"),
      json: vi.fn(),
    });
    vi.stubGlobal("fetch", fetchSpy);

    const { fetchFromBackend } = await import("./api");

    await expect(
      fetchFromBackend("jobs/42/logs", { parseAs: "text" }),
    ).resolves.toBe("job log output");
  });

  it("throws when the backend response is not ok", async () => {
    process.env.BACKEND_API_URL = "https://backend.example.test";
    const fetchSpy = vi.fn().mockResolvedValue({
      ok: false,
      status: 503,
      text: vi.fn().mockResolvedValue("backend unavailable"),
    });
    vi.stubGlobal("fetch", fetchSpy);

    const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const { fetchFromBackend } = await import("./api");

    await expect(fetchFromBackend("jobs/42")).rejects.toThrow(
      "Backend responded with status: 503",
    );
    expect(consoleErrorSpy).toHaveBeenCalledWith(
      "[Backend API Error]",
      expect.objectContaining({
        status: 503,
        body: "backend unavailable",
      }),
    );
  });
});
