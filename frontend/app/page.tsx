
"use client";

import { SubmitEvent, useState } from "react";

type TrainingRun = {
  name: string;
  namespace: string;
  status: string;
};

export default function Home() {
  const [name, setName] = useState("experiment-001");
  const [epochs, setEpochs] = useState(100);
  const [run, setRun] = useState<TrainingRun | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  async function handleSubmit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError("");
    setRun(null);

    try {
      const response = await fetch(
        "http://localhost:8080/api/v1/training-runs",
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({ name, epochs }),
        }
      );

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.message ?? data.error ?? "Failed to create training run");
      }

      setRun(data);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "An unexpected error occurred"
      );
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="min-h-screen bg-slate-950 px-6 py-16 text-slate-100">
      <div className="mx-auto max-w-3xl">
        <header className="mb-10">
          <p className="text-sm font-medium tracking-widest text-blue-400">
            KUBEAI
          </p>
          <h1 className="mt-3 text-4xl font-bold tracking-tight">
            AI Training Platform
          </h1>
          <p className="mt-3 text-slate-400">
            Submit and manage your model training experiments.
          </p>
        </header>

        <section className="rounded-2xl border border-slate-800 bg-slate-900 p-6">
          <h2 className="text-xl font-semibold">New Training Run</h2>
          <p className="mt-2 text-sm text-slate-400">
            Configure your experiment and submit it to Kubernetes.
          </p>

          <form onSubmit={handleSubmit} className="mt-6 space-y-5">
            <div>
              <label
                htmlFor="name"
                className="mb-2 block text-sm font-medium"
              >
                Experiment name
              </label>
              <input
                id="name"
                value={name}
                onChange={(event) => setName(event.target.value)}
                required
                pattern="[a-z0-9]([-a-z0-9]*[a-z0-9])?"
                title="Use lowercase letters, numbers, and hyphens."
                className="w-full rounded-lg border border-slate-700 bg-slate-950 px-4 py-3 outline-none focus:border-blue-500"
              />
            </div>

            <div>
              <label
                htmlFor="epochs"
                className="mb-2 block text-sm font-medium"
              >
                Training epochs
              </label>
              <input
                id="epochs"
                type="number"
                min={1}
                max={10000}
                value={epochs}
                onChange={(event) => setEpochs(Number(event.target.value))}
                required
                className="w-full rounded-lg border border-slate-700 bg-slate-950 px-4 py-3 outline-none focus:border-blue-500"
              />
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full rounded-lg bg-blue-600 px-4 py-3 font-semibold transition hover:bg-blue-500 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {loading ? "Submitting..." : "Start Training"}
            </button>
          </form>

          {error && (
            <div
              role="alert"
              className="mt-5 rounded-lg border border-red-800 bg-red-950 p-4 text-sm text-red-300"
            >
              {error}
            </div>
          )}

          {run && (
            <div
              role="status"
              className="mt-6 rounded-xl border border-emerald-800 bg-emerald-950/40 p-5"
            >
              <p className="font-semibold text-emerald-400">
                Training run submitted successfully
              </p>
              <dl className="mt-4 space-y-3 text-sm">
                <div className="flex justify-between gap-4">
                  <dt className="text-slate-400">Name</dt>
                  <dd className="font-mono">{run.name}</dd>
                </div>
                <div className="flex justify-between gap-4">
                  <dt className="text-slate-400">Namespace</dt>
                  <dd>{run.namespace}</dd>
                </div>
                <div className="flex justify-between gap-4">
                  <dt className="text-slate-400">Status</dt>
                  <dd>{run.status}</dd>
                </div>
              </dl>
            </div>
          )}
        </section>

        <footer className="mt-8 text-center text-xs text-slate-500">
          Powered by Go, Kubernetes, and PyTorch
        </footer>
      </div>
    </main>
  );
}
