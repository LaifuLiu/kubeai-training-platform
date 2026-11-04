"use client";

import { TrainingRuns } from "@/types/training-runs";
import { useQuery } from "@tanstack/react-query";
import { UsePaginationOptions } from "./use-pagination";

async function fetchTrainingRuns(params?: UsePaginationOptions) {
  const res = await fetch(
    `/api/v1/training-runs?page=${params?.initialPage ?? 1}&size=${params?.initialSize ?? 10}`,
  );
  if (!res.ok) throw new Error("Failed to fetch live training runs");
  return res.json();
}

export function useTrainingRuns(params?: UsePaginationOptions) {
  return useQuery<TrainingRuns>({
    queryKey: [
      "get",
      "api/v1",
      "training-runs",
      params?.initialPage ?? 1,
      params?.initialSize ?? 10,
    ],
    queryFn: () => fetchTrainingRuns(params),
    refetchInterval: 5000, // <--- Polls the backend every 5 seconds automatically
  });
}
