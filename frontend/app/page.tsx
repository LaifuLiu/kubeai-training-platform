
"use client";

import { useCallback, useEffect, useState } from "react";

const API = "http://localhost:8080/api/v1/training-runs";

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
  const [name, setName] = useState("");
  const [epochs, setEpochs] = useState(10);
  const [runs, setRuns] = useState<TrainingRun[]>([]);
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const [selectedRun, setSelectedRun] = useState("");
  const [logs, setLogs] = useState("");
  const [logsLoading, setLogsLoading] = useState(false);
  const [logsError, setLogsError] = useState("");

  const [cancelTarget, setCancelTarget] = useState("");
  const [cancelLoading, setCancelLoading] = useState(false);
  const [cancelError, setCancelError] = useState("");
  const [cancelNotice, setCancelNotice] = useState("");

  const [cpu, setCpu] = useState("500m");
  const [memory, setMemory] = useState("1Gi");

  const loadRuns = useCallback(async () => {
    try {
      const response = await fetch(API, { cache: "no-store" });
      if (!response.ok) {
        throw new Error(`Failed to load training runs (${response.status})`);
      }
      setRuns(await response.json());
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to load training runs"
      );
    }
  }, []);

  const loadLogs = useCallback(async (runName: string) => {
    setSelectedRun(runName);
    setLogs("");
    setLogsError("");
    setLogsLoading(true);

    try {
      const response = await fetch(
        `${API}/${encodeURIComponent(runName)}/logs`,
        { cache: "no-store" }
      );

      const text = await response.text();

      if (!response.ok) {
        throw new Error(text || `Failed to load logs (${response.status})`);
      }

      setLogs(text || "No logs have been produced yet.");
    } catch (err) {
      setLogsError(
        err instanceof Error ? err.message : "Failed to load training logs"
      );
    } finally {
      setLogsLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadRuns();
    const timer = window.setInterval(() => {
      void loadRuns();
    }, 5000);

    return () => window.clearInterval(timer);
  }, [loadRuns]);

  async function createRun(event: React.SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    setNotice("");
    setError("");
    setLoading(true);

    try {
      const response = await fetch(API, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: name.trim(), epochs, cpu, memory }),
      });

      const result = await response.json();

      if (!response.ok) {
        throw new Error(
          typeof result === "string"
            ? result
            : result.error || "Failed to create training run"
        );
      }

      setNotice(`Training run "${result.name}" submitted successfully.`);
      setName("");
      await loadRuns();
      await loadLogs(result.name);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to create training run"
      );
    } finally {
      setLoading(false);
    }
  }


  async function cancelRun(runName: string) {
    setCancelError("");
    setCancelNotice("");
    setCancelLoading(true);

    try {
      const response = await fetch(
        `${API}/${encodeURIComponent(runName)}`,
        { method: "DELETE" }
      );

      const message = await response.text();

      if (!response.ok) {
        throw new Error(
          message || `Failed to cancel training run (${response.status})`
        );
      }

      setCancelNotice(
        `Deletion requested for "${runName}". Kubernetes is terminating its resources.`
      );
      setCancelTarget("");

      if (selectedRun === runName) {
        setSelectedRun("");
        setLogs("");
        setLogsError("");
      }

      await loadRuns();
    } catch (err) {
      setCancelError(
        err instanceof Error ? err.message : "Failed to cancel training run"
      );
    } finally {
      setCancelLoading(false);
    }
  }

  return (
    <main className="min-h-screen bg-slate-950 px-6 py-10 text-slate-100">
      <div className="mx-auto max-w-6xl space-y-8">
        <header>
          <p className="text-sm font-medium uppercase tracking-widest text-cyan-400">
            KubeAI Platform
          </p>
          <h1 className="mt-2 text-3xl font-bold">
            AI Training Dashboard
          </h1>
          <p className="mt-2 text-slate-400">
            Submit, monitor, and inspect Kubernetes training runs.
          </p>
        </header>

        <section className="rounded-xl border border-slate-800 bg-slate-900 p-6">
          <h2 className="text-xl font-semibold">Create training run</h2>

          <form
            onSubmit={createRun}
            className="mt-5 flex flex-wrap items-end gap-4"
          >
            <label className="flex min-w-56 flex-1 flex-col gap-2 text-sm">
              Experiment name
              <input
                required
                pattern="[a-z0-9]([-a-z0-9]*[a-z0-9])?"
                minLength={1}
                maxLength={63}
                value={name}
                onChange={(event) => setName(event.target.value)}
                placeholder="my-training-run"
                className="rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 outline-none focus:border-cyan-500"
              />
              <span className="text-xs text-slate-500">
                Lowercase letters, numbers, and hyphens.
              </span>
            </label>

            <label className="flex w-36 flex-col gap-2 text-sm">
              Epochs
              <input
                required
                type="number"
                min={1}
                max={1000}
                value={epochs}
                onChange={(event) => setEpochs(Number(event.target.value))}
                className="rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 outline-none focus:border-cyan-500"
              />
            </label>

            <label className="flex w-36 flex-col gap-2 text-sm">
              CPU
              <input
                value={cpu}
                onChange={(e) => setCpu(e.target.value)}
                placeholder="500m"
                className="rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 outline-none focus:border-cyan-500"
              />
            </label>

            <label className="flex w-36 flex-col gap-2 text-sm">
              Memory
              <input
                value={memory}
                onChange={(e) => setMemory(e.target.value)}
                placeholder="1Gi"
                className="rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 outline-none focus:border-cyan-500"
              />
            </label>

            <button
              type="submit"
              disabled={loading}
              className="rounded-lg bg-cyan-500 px-5 py-2.5 font-semibold text-slate-950 hover:bg-cyan-400 disabled:opacity-50"
            >
              {loading ? "Submitting..." : "Start training"}
            </button>
          </form>

          {notice && (
            <p role="status" className="mt-4 text-sm text-emerald-400">
              {notice}
            </p>
          )}
          {error && (
            <p role="alert" className="mt-4 whitespace-pre-wrap text-sm text-red-400">
              {error}
            </p>
          )}
        </section>

        <section className="overflow-hidden rounded-xl border border-slate-800 bg-slate-900">
          <div className="flex items-center justify-between border-b border-slate-800 p-6">
            <div>
              <h2 className="text-xl font-semibold">Training runs</h2>
              <p className="mt-1 text-sm text-slate-400">
                Refreshes automatically every 5 seconds.
              </p>
            </div>
            <button
              onClick={() => void loadRuns()}
              className="rounded-lg border border-slate-700 px-4 py-2 text-sm hover:bg-slate-800"
            >
              Refresh
            </button>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="bg-slate-950 text-slate-400">
                <tr>
                  <th className="px-6 py-3">Name</th>
                  <th className="px-6 py-3">Epochs</th>
                  <th className="px-6 py-3">Status</th>
                  <th className="px-6 py-3">Pods (A/S/F)</th>
                  <th className="px-6 py-3">Created</th>
                  <th className="px-6 py-3">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800">
                {runs.map((run) => (
                  <tr key={run.name} className="hover:bg-slate-800/50">
                    <td className="px-6 py-4 font-medium">{run.name}</td>
                    <td className="px-6 py-4">{run.epochs ?? "—"}</td>
                    <td className="px-6 py-4">
                      <span className="rounded-full bg-slate-800 px-2.5 py-1 text-xs">
                        {run.status}
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      {run.active}/{run.succeeded}/{run.failed}
                    </td>
                    <td className="whitespace-nowrap px-6 py-4 text-slate-400">
                      {run.createdAt
                        ? new Date(run.createdAt).toLocaleString()
                        : "—"}
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => void loadLogs(run.name)}
                          className="whitespace-nowrap rounded-lg border border-cyan-700 px-3 py-1.5 text-cyan-300 hover:bg-cyan-950"
                        >
                          View logs
                        </button>

                        <button
                          onClick={() => {
                            setCancelError("");
                            setCancelNotice("");
                            setCancelTarget(run.name);
                          }}
                          disabled={cancelLoading}
                          className="whitespace-nowrap rounded-lg border border-red-800 px-3 py-1.5 text-red-300 hover:bg-red-950 disabled:opacity-50"
                        >
                          Cancel
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}

                {runs.length === 0 && (
                  <tr>
                    <td colSpan={6} className="px-6 py-10 text-center text-slate-500">
                      No training runs found. Create your first run above.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </section>


        {cancelTarget && (
          <section
            role="dialog"
            aria-modal="true"
            aria-labelledby="cancel-title"
            className="rounded-xl border border-red-900 bg-slate-900 p-6"
          >
            <h2 id="cancel-title" className="text-lg font-semibold">
              Cancel training run?
            </h2>

            <p className="mt-2 text-sm text-slate-300">
              This will delete the Kubernetes Job{" "}
              <strong>{cancelTarget}</strong> and request deletion of its dependent
              Pods. This action cannot be undone.
            </p>

            <div className="mt-5 flex flex-wrap gap-3">
              <button
                onClick={() => void cancelRun(cancelTarget)}
                disabled={cancelLoading}
                className="rounded-lg bg-red-600 px-4 py-2 font-semibold text-white hover:bg-red-500 disabled:opacity-50"
              >
                {cancelLoading ? "Cancelling..." : "Confirm cancellation"}
              </button>

              <button
                onClick={() => setCancelTarget("")}
                disabled={cancelLoading}
                className="rounded-lg border border-slate-700 px-4 py-2 hover:bg-slate-800 disabled:opacity-50"
              >
                Keep run
              </button>
            </div>
          </section>
        )}

        {cancelNotice && (
          <p role="status" className="text-sm text-emerald-400">
            {cancelNotice}
          </p>
        )}

        {cancelError && (
          <p role="alert" className="whitespace-pre-wrap text-sm text-red-400">
            {cancelError}
          </p>
        )}


        <section className="overflow-hidden rounded-xl border border-slate-800 bg-slate-900">
          <div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-800 p-6">
            <div>
              <h2 className="text-xl font-semibold">Training logs</h2>
              <p className="mt-1 text-sm text-slate-400">
                {selectedRun
                  ? `Run: ${selectedRun}`
                  : "Select a training run to inspect its output."}
              </p>
            </div>

            {selectedRun && (
              <button
                onClick={() => void loadLogs(selectedRun)}
                disabled={logsLoading}
                className="rounded-lg border border-slate-700 px-4 py-2 text-sm hover:bg-slate-800 disabled:opacity-50"
              >
                {logsLoading ? "Loading..." : "Reload logs"}
              </button>
            )}
          </div>

          <div className="min-h-56 bg-black p-5">
            {logsLoading ? (
              <p className="text-sm text-slate-400">Loading logs...</p>
            ) : logsError ? (
              <pre className="whitespace-pre-wrap break-words text-sm text-red-400">
                {logsError}
              </pre>
            ) : logs ? (
              <pre className="max-h-[32rem] overflow-auto whitespace-pre-wrap break-words font-mono text-sm leading-6 text-emerald-300">
                {logs}
              </pre>
            ) : (
              <p className="text-sm text-slate-500">
                Training output will appear here when you select a run.
              </p>
            )}
          </div>
        </section>
      </div>
    </main>
  );
}
