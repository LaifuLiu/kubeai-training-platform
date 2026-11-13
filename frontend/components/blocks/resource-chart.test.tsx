import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { useNodes } from "@/hooks/query-nodes";
import { ResourceChart, formatKiMemory } from "./resource-chart";

vi.mock("@/hooks/query-nodes", () => ({
  useNodes: vi.fn(),
}));

describe("ResourceChart", () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("formats Ki memory values into Mi/Gi units", () => {
    expect(formatKiMemory("1024")).toBe("1.0Mi");
    expect(formatKiMemory("1048576")).toBe("1.0Gi");
    expect(formatKiMemory("500")).toBe("500Ki");
  });

  it("renders the node data and normalized memory strings", () => {
    vi.mocked(useNodes).mockReturnValue({
      data: [
        {
          name: "worker-1",
          ready: true,
          unschedulable: false,
          capacity: {
            cpu: "2000m",
            memory: "2097152Ki",
            gpus: "2",
          },
          allocatable: {
            cpu: "1000m",
            memory: "1048576Ki",
            gpus: "1",
          },
        },
      ],
      isLoading: false,
      error: null,
    } as any);

    render(<ResourceChart />);

    expect(screen.getByText("Node Resource Usage")).toBeInTheDocument();
    expect(screen.getByText("1000m / 2000m")).toBeInTheDocument();
    expect(screen.getByText("1.0Gi / 2.0Gi")).toBeInTheDocument();
    expect(screen.getByText("1 / 2")).toBeInTheDocument();
  });

  it("shows a loading state while nodes are loading", () => {
    vi.mocked(useNodes).mockReturnValue({
      data: undefined,
      isLoading: true,
      error: null,
    } as any);

    render(<ResourceChart />);

    expect(screen.getByText("Loading...")).toBeInTheDocument();
  });

  it("shows an error state when node fetching fails", () => {
    vi.mocked(useNodes).mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error("Failed to fetch live nodes"),
    } as any);

    render(<ResourceChart />);

    expect(screen.getByText("Error loading nodes")).toBeInTheDocument();
  });
});
