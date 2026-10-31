"use client";

import { useCallback, useEffect, useState } from "react";

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

export default function Home() {
  return <div>Home</div>;
  // const [name, setName] = useState("");
  // const [epochs, setEpochs] = useState(10);
  // const [runs, setRuns] = useState<TrainingRun[]>([]);
  // const [notice, setNotice] = useState("");
  // const [error, setError] = useState("");
  // const [loading, setLoading] = useState(false);

  // const [selectedRun, setSelectedRun] = useState("");
  // const [logs, setLogs] = useState("");
  // const [logsLoading, setLogsLoading] = useState(false);
  // const [logsError, setLogsError] = useState("");

  // const [cancelTarget, setCancelTarget] = useState("");
  // const [cancelLoading, setCancelLoading] = useState(false);
  // const [cancelError, setCancelError] = useState("");
  // const [cancelNotice, setCancelNotice] = useState("");

  // const [cpu, setCpu] = useState("500m");
  // const [memory, setMemory] = useState("1Gi");
  // const [priority, setPriority] = useState("normal");
  // const [replicas, setReplicas] = useState(1);

  // const loadLogs = useCallback(async (runName: string) => {
  //   setSelectedRun(runName);
  //   setLogs("");
  //   setLogsError("");
  //   setLogsLoading(true);

  //   try {
  //     const response = await fetch(
  //       `${API}/${encodeURIComponent(runName)}/logs`,
  //       { cache: "no-store" },
  //     );

  //     const text = await response.text();

  //     if (!response.ok) {
  //       throw new Error(text || `Failed to load logs (${response.status})`);
  //     }

  //     setLogs(text || "No logs have been produced yet.");
  //   } catch (err) {
  //     setLogsError(
  //       err instanceof Error ? err.message : "Failed to load training logs",
  //     );
  //   } finally {
  //     setLogsLoading(false);
  //   }
  // }, []);

  // useEffect(() => {
  //   void loadRuns();
  //   const timer = window.setInterval(() => {
  //     void loadRuns();
  //   }, 5000);

  //   return () => window.clearInterval(timer);
  // }, [loadRuns]);

  // async function createRun(event: React.SubmitEvent<HTMLFormElement>) {
  //   event.preventDefault();
  //   setNotice("");
  //   setError("");
  //   setLoading(true);

  //   try {
  //     const response = await fetch(API, {
  //       method: "POST",
  //       headers: { "Content-Type": "application/json" },
  //       body: JSON.stringify({
  //         name: name.trim(),
  //         epochs,
  //         cpu,
  //         memory,
  //         priority,
  //         replicas,
  //       }),
  //     });

  //     const result = await response.json();

  //     if (!response.ok) {
  //       throw new Error(
  //         typeof result === "string"
  //           ? result
  //           : result.error || "Failed to create training run",
  //       );
  //     }

  //     setNotice(`Training run "${result.name}" submitted successfully.`);
  //     setName("");
  //     await loadRuns();
  //     await loadLogs(result.name);
  //   } catch (err) {
  //     setError(
  //       err instanceof Error ? err.message : "Failed to create training run",
  //     );
  //   } finally {
  //     setLoading(false);
  //   }
  // }

  // async function cancelRun(runName: string) {
  //   setCancelError("");
  //   setCancelNotice("");
  //   setCancelLoading(true);

  //   try {
  //     const response = await fetch(`${API}/${encodeURIComponent(runName)}`, {
  //       method: "DELETE",
  //     });

  //     const message = await response.text();

  //     if (!response.ok) {
  //       throw new Error(
  //         message || `Failed to cancel training run (${response.status})`,
  //       );
  //     }

  //     setCancelNotice(
  //       `Deletion requested for "${runName}". Kubernetes is terminating its resources.`,
  //     );
  //     setCancelTarget("");

  //     if (selectedRun === runName) {
  //       setSelectedRun("");
  //       setLogs("");
  //       setLogsError("");
  //     }

  //     await loadRuns();
  //   } catch (err) {
  //     setCancelError(
  //       err instanceof Error ? err.message : "Failed to cancel training run",
  //     );
  //   } finally {
  //     setCancelLoading(false);
  //   }
  // }

  // return <div>home</div>;
}
