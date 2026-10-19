import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import Home from "./page";

type TrainingRun = {
  name: string;
  namespace: string;
  status: string;
  epochs?: number;
  createdAt: string;
  active: number;
  succeeded: number;
  failed: number;
};

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
    text: async () =>
      typeof body === "string" ? body : JSON.stringify(body),
  } as Response;
}

function textResponse(body: string, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
    text: async () => body,
  } as Response;
}

const sampleRun: TrainingRun = {
  name: "mnist",
  namespace: "kubeai-training",
  status: "Running",
  epochs: 10,
  createdAt: "2026-10-19T12:00:00.000Z",
  active: 1,
  succeeded: 0,
  failed: 0,
};

function mockApi(options?: {
  list?: () => Promise<Response>;
  create?: (init?: RequestInit) => Promise<Response>;
  logs?: (runName: string) => Promise<Response>;
  cancel?: (runName: string) => Promise<Response>;
}) {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    const method = (init?.method ?? "GET").toUpperCase();

    if (url.endsWith("/logs") && method === "GET") {
      const runName = decodeURIComponent(url.split("/").at(-2) ?? "");
      return (options?.logs ?? (() => Promise.resolve(textResponse("epoch 1"))))(
        runName
      );
    }

    if (url.endsWith("/training-runs") && method === "GET") {
      return (options?.list ?? (() => Promise.resolve(jsonResponse([]))))();
    }

    if (url.endsWith("/training-runs") && method === "POST") {
      return (
        options?.create ??
        (() =>
          Promise.resolve(
            jsonResponse(
              { name: "mnist", namespace: "kubeai-training", status: "submitted" },
              201
            )
          ))
      )(init);
    }

    if (method === "DELETE") {
      const runName = decodeURIComponent(url.split("/").at(-1) ?? "");
      return (
        options?.cancel ??
        (() =>
          Promise.resolve(
            textResponse(JSON.stringify({ status: "deletion_requested" }))
          ))
      )(runName);
    }

    throw new Error(`Unhandled fetch ${method} ${url}`);
  });

  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

describe("Home dashboard", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("renders the dashboard chrome", async () => {
    mockApi();
    render(<Home />);

    expect(
      screen.getByRole("heading", { level: 1, name: "AI Training Dashboard" })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Create training run" })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Training runs" })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Training logs" })
    ).toBeInTheDocument();

    await waitFor(() => {
      expect(
        screen.getByText("No training runs found. Create your first run above.")
      ).toBeInTheDocument();
    });
  });

  it("shows an error when listing runs fails", async () => {
    mockApi({
      list: async () => jsonResponse("nope", 500),
    });

    render(<Home />);

    expect(
      await screen.findByRole("alert")
    ).toHaveTextContent("Failed to load training runs (500)");
  });

  it("lists training runs and allows a manual refresh", async () => {
    const user = userEvent.setup();
    let listCalls = 0;
    const fetchMock = mockApi({
      list: async () => {
        listCalls += 1;
        return jsonResponse(listCalls === 1 ? [sampleRun] : []);
      },
    });

    render(<Home />);

    expect(await screen.findByText("mnist")).toBeInTheDocument();
    expect(screen.getByText("Running")).toBeInTheDocument();
    expect(screen.getByText("1/0/0")).toBeInTheDocument();
    expect(screen.getByText("10")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Refresh" }));

    await waitFor(() => {
      expect(
        screen.getByText("No training runs found. Create your first run above.")
      ).toBeInTheDocument();
    });
    expect(fetchMock).toHaveBeenCalled();
  });

  it("renders placeholders when epochs and createdAt are missing", async () => {
    mockApi({
      list: async () =>
        jsonResponse([
          {
            name: "partial",
            namespace: "kubeai-training",
            status: "Pending",
            createdAt: "",
            active: 0,
            succeeded: 0,
            failed: 0,
          },
        ]),
    });

    render(<Home />);

    const row = (await screen.findByText("partial")).closest("tr");
    expect(row).not.toBeNull();
    expect(within(row as HTMLElement).getAllByText("—").length).toBeGreaterThan(0);
  });

  it("submits a training run with form values and loads its logs", async () => {
    const user = userEvent.setup();
    const fetchMock = mockApi({
      list: async () => jsonResponse([]),
      create: async () =>
        jsonResponse(
          { name: "resnet", namespace: "kubeai-training", status: "submitted" },
          201
        ),
      logs: async () => textResponse("training started"),
    });

    render(<Home />);
    await screen.findByText("No training runs found. Create your first run above.");

    await user.type(screen.getByPlaceholderText("my-training-run"), "resnet");
    await user.clear(screen.getByLabelText("Epochs"));
    await user.type(screen.getByLabelText("Epochs"), "20");
    await user.clear(screen.getByPlaceholderText("500m"));
    await user.type(screen.getByPlaceholderText("500m"), "1");
    await user.clear(screen.getByPlaceholderText("1Gi"));
    await user.type(screen.getByPlaceholderText("1Gi"), "2Gi");
    await user.selectOptions(screen.getByLabelText("Priority"), "high");
    await user.clear(screen.getByLabelText("Replicas"));
    await user.type(screen.getByLabelText("Replicas"), "2");

    await user.click(screen.getByRole("button", { name: "Start training" }));

    expect(
      await screen.findByRole("status")
    ).toHaveTextContent('Training run "resnet" submitted successfully.');
    expect(await screen.findByText("training started")).toBeInTheDocument();
    expect(screen.getByText("Run: resnet")).toBeInTheDocument();

    const createCall = fetchMock.mock.calls.find(
      ([, init]) => init?.method === "POST"
    );
    expect(createCall).toBeDefined();
    expect(JSON.parse(String(createCall?.[1]?.body))).toEqual({
      name: "resnet",
      epochs: 20,
      cpu: "1",
      memory: "2Gi",
      priority: "high",
      replicas: 2,
    });
  });

  it("shows a create error from the API", async () => {
    const user = userEvent.setup();
    mockApi({
      create: async () => jsonResponse({ error: "name is required" }, 400),
    });

    render(<Home />);
    await waitFor(() => expect(fetch).toHaveBeenCalled());

    await user.type(screen.getByPlaceholderText("my-training-run"), "bad-run");
    await user.click(screen.getByRole("button", { name: "Start training" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("name is required");
  });

  it("loads logs for a selected run and reloads them", async () => {
    const user = userEvent.setup();
    let logCalls = 0;
    mockApi({
      list: async () => jsonResponse([sampleRun]),
      logs: async () => {
        logCalls += 1;
        return textResponse(logCalls === 1 ? "first log" : "second log");
      },
    });

    render(<Home />);
    await user.click(await screen.findByRole("button", { name: "View logs" }));

    expect(await screen.findByText("first log")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Reload logs" }));
    expect(await screen.findByText("second log")).toBeInTheDocument();
  });

  it("shows a fallback when logs are empty", async () => {
    const user = userEvent.setup();
    mockApi({
      list: async () => jsonResponse([sampleRun]),
      logs: async () => textResponse(""),
    });

    render(<Home />);
    await user.click(await screen.findByRole("button", { name: "View logs" }));

    expect(
      await screen.findByText("No logs have been produced yet.")
    ).toBeInTheDocument();
  });

  it("shows a logs error", async () => {
    const user = userEvent.setup();
    mockApi({
      list: async () => jsonResponse([sampleRun]),
      logs: async () => textResponse("pod not ready", 404),
    });

    render(<Home />);
    await user.click(await screen.findByRole("button", { name: "View logs" }));

    expect(await screen.findByText("pod not ready")).toBeInTheDocument();
  });

  it("opens the cancel dialog and can keep the run", async () => {
    const user = userEvent.setup();
    mockApi({
      list: async () => jsonResponse([sampleRun]),
    });

    render(<Home />);
    await user.click(await screen.findByRole("button", { name: "Cancel" }));

    const dialog = screen.getByRole("dialog");
    expect(dialog).toHaveTextContent("Cancel training run?");
    expect(dialog).toHaveTextContent("mnist");

    await user.click(screen.getByRole("button", { name: "Keep run" }));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("confirms cancellation and clears selected logs for that run", async () => {
    const user = userEvent.setup();
    const fetchMock = mockApi({
      list: async () => jsonResponse([sampleRun]),
      logs: async () => textResponse("alive"),
    });

    render(<Home />);
    await user.click(await screen.findByRole("button", { name: "View logs" }));
    expect(await screen.findByText("alive")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Cancel" }));
    await user.click(
      screen.getByRole("button", { name: "Confirm cancellation" })
    );

    expect(await screen.findByRole("status")).toHaveTextContent(
      'Deletion requested for "mnist"'
    );
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(
      screen.getByText("Select a training run to inspect its output.")
    ).toBeInTheDocument();
    expect(
      screen.getByText("Training output will appear here when you select a run.")
    ).toBeInTheDocument();

    expect(
      fetchMock.mock.calls.some(
        ([url, init]) =>
          String(url).endsWith("/mnist") && init?.method === "DELETE"
      )
    ).toBe(true);
  });

  it("shows a cancel error", async () => {
    const user = userEvent.setup();
    mockApi({
      list: async () => jsonResponse([sampleRun]),
      cancel: async () => textResponse("training run is not managed by KubeAI", 403),
    });

    render(<Home />);
    await user.click(await screen.findByRole("button", { name: "Cancel" }));
    await user.click(
      screen.getByRole("button", { name: "Confirm cancellation" })
    );

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "training run is not managed by KubeAI"
    );
  });

  it("clears the poll timer on unmount", async () => {
    mockApi();
    const clearSpy = vi.spyOn(window, "clearInterval");
    const { unmount } = render(<Home />);
    await waitFor(() => expect(fetch).toHaveBeenCalled());

    unmount();
    expect(clearSpy).toHaveBeenCalled();
  });
});
