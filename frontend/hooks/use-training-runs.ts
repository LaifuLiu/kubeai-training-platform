"use client";

import { TrainingRuns } from "@/types/training-runs";
import { useQuery } from "@tanstack/react-query";

async function fetchTrainingRuns() {
  const res = await fetch("/api/v1/training-runs");
  if (!res.ok) throw new Error("Failed to fetch live training runs");
  return res.json();
}

export function useTrainingRuns() {
  return useQuery<TrainingRuns[]>({
    queryKey: ["get", "api/v1", "training-runs"],
    queryFn: fetchTrainingRuns,
    refetchInterval: 5000, // <--- Polls the backend every 5 seconds automatically
  });
}
