"use client";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useTrainingRuns } from "@/hooks/use-training-runs";

export default function Page() {
  const { status, data, refetch } = useTrainingRuns();

  return (
    <div>
      <section className="overflow-hidden">
        <div>
          <h2 className="text-xl font-semibold">Training runs</h2>
          <p className="mt-1 text-sm">
            Refreshes automatically every 5 seconds.
          </p>
        </div>
        <div className="flex items-center justify-between border-b border-slate-800 p-6">
          <button
            onClick={() => {
              refetch();
            }}
            className="rounded-lg border border-slate-700 px-4 py-2 text-sm hover:bg-slate-800"
          >
            Refresh
          </button>
        </div>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Epochs</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Pods (A/S/F)</TableHead>
              <TableHead>Created</TableHead>
              <TableHead>Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data &&
              data.map((run) => (
                <TableRow key={run.name}>
                  <TableCell>{run.name}</TableCell>
                  <TableCell>{run.epochs ?? "—"}</TableCell>
                  <TableCell>
                    <span>{run.status}</span>
                  </TableCell>
                  <TableCell>
                    {run.active}/{run.succeeded}/{run.failed}
                  </TableCell>
                  <TableCell>
                    {run.createdAt
                      ? new Date(run.createdAt).toLocaleString()
                      : "—"}
                  </TableCell>
                  {/* <TableCell>
                    <div className="flex items-center gap-2">
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
                  </TableCell> */}
                </TableRow>
              ))}

            {data && data.length === 0 && (
              <TableRow>
                <TableCell
                  colSpan={6}
                  className="px-6 py-10 text-center text-slate-500"
                >
                  No training runs found. Create your first run above.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </section>

      {/* {cancelTarget && (
        <section
          role="dialog"
          aria-modal="true"
          aria-labelledby="cancel-title"
          className="rounded-xl border border-red-900 bg-slate-900 p-6"
        >
          <h2 id="cancel-title" className="text-lg font-semibold">
            Cancel training run?
          </h2> */}

      {/* <p className="mt-2 text-sm text-slate-300">
            This will delete the Kubernetes Job <strong>{cancelTarget}</strong>{" "}
            and request deletion of its dependent Pods. This action cannot be
            undone.
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
          </div> */}

      {/* {cancelNotice && (
        <p role="status" className="text-sm text-emerald-400">
          {cancelNotice}
        </p>
      )}

      {cancelError && (
        <p role="alert" className="whitespace-pre-wrap text-sm text-red-400">
          {cancelError}
        </p>
      )} */}

      {/* <section className="overflow-hidden rounded-xl border border-slate-800 bg-slate-900">
            <div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-800 p-6">
              <div>
                <h2 className="text-xl font-semibold"></h2>
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
                <pre className="whitespace-pre-wrap wrap-break-wordword text-sm text-red-400">
                  {logsError}
                </pre>
              ) : logs ? (
                <pre className="max-h-128 overflow-auto whitespace-pre-wrap wrap-break-word font-mono text-sm leading-6 text-emerald-300">
                  {logs}
                </pre>
              ) : (
                <p className="text-sm text-slate-500">
                  Training output will appear here when you select a run.
                </p>
              )}
            </div>
          </section> */}
    </div>
  );
}
