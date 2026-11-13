"use client";

import { useMutation } from "@tanstack/react-query";
import { TrainingRunRequest } from "@/components/blocks/form";

async function postTrainingRuns(data: TrainingRunRequest) {
  const res = await fetch(`/api/v1/training-runs`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(data),
  });
  if (!res.ok) throw new Error("Failed to create live training runs");
  return res.json();
}

export function useMutateTrainingRuns() {
  return useMutation({
    mutationKey: ["post", "api/v1", "training-runs"],
    mutationFn: (data: TrainingRunRequest) => postTrainingRuns(data),
  });
}
