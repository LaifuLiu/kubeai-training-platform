"use client";

import { Nodes } from "@/types/nodes";
import { useQuery } from "@tanstack/react-query";

async function fetchNodes() {
  const res = await fetch("/api/v1/nodes");
  if (!res.ok) throw new Error("Failed to fetch live nodes");
  return res.json();
}

export function useNodes() {
  return useQuery<Nodes[]>({
    queryKey: ["get", "api/v1", "nodes"],
    queryFn: fetchNodes,
    refetchInterval: 5000, // <--- Polls the backend every 5 seconds automatically
  });
}
