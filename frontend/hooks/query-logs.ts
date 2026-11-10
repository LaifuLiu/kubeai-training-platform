"use client";

import { useQuery } from "@tanstack/react-query";

async function fetchLogs({ slug }: { slug: string }) {
  const res = await fetch(`/api/v1/training-runs/${slug}/logs`);
  if (!res.ok) throw new Error("Failed to fetch logs");
  return res.text();
}

export function useLogs({ slug }: { slug: string }) {
  return useQuery({
    queryKey: ["get", "api/v1", "training-runs", slug, "logs"],
    queryFn: () => fetchLogs({ slug }),
    refetchInterval: 5000, // <--- Polls the backend every 5 seconds automatically
  });
}
