"use client";
import { use } from "react";
import { BreadcrumbBlock } from "@/components/blocks/breadcrumb-block";
import { Separator } from "@/components/ui/separator";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Logs } from "@/components/blocks/logs";

export default function TrainingJobPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = use(params);

  return (
    <div>
      <BreadcrumbBlock slug={slug} />
      <h1 className="font-extrabold mt-4 mb-8 text-3xl">{slug}</h1>
      <Separator className="my-4" />
      <div>
        <Card>
          <CardHeader>
            <CardTitle>Training Logs</CardTitle>
          </CardHeader>
          <CardContent>
            <Logs slug={slug} />
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

{
  /* <TableCell>
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
                  </TableCell> */
}

//     {/* {cancelTarget && (
//       <section
//         role="dialog"
//         aria-modal="true"
//         aria-labelledby="cancel-title"
//         className="rounded-xl border border-red-900 bg-slate-900 p-6"
//       >
//         <h2 id="cancel-title" className="text-lg font-semibold">
//           Cancel training run?
//         </h2> */}

//     {/* <p className="mt-2 text-sm text-slate-300">
//           This will delete the Kubernetes Job <strong>{cancelTarget}</strong>{" "}
//           and request deletion of its dependent Pods. This action cannot be
//           undone.
//         </p>

//         <div className="mt-5 flex flex-wrap gap-3">
//           <button
//             onClick={() => void cancelRun(cancelTarget)}
//             disabled={cancelLoading}
//             className="rounded-lg bg-red-600 px-4 py-2 font-semibold text-white hover:bg-red-500 disabled:opacity-50"
//           >
//             {cancelLoading ? "Cancelling..." : "Confirm cancellation"}
//           </button>

//           <button
//             onClick={() => setCancelTarget("")}
//             disabled={cancelLoading}
//             className="rounded-lg border border-slate-700 px-4 py-2 hover:bg-slate-800 disabled:opacity-50"
//           >
//             Keep run
//           </button>
//         </div> */}

//     {/* {cancelNotice && (
//       <p role="status" className="text-sm text-emerald-400">
//         {cancelNotice}
//       </p>
//     )}

//     {cancelError && (
//       <p role="alert" className="whitespace-pre-wrap text-sm text-red-400">
//         {cancelError}
//       </p>
//     )} */}

//     {/* <section className="overflow-hidden rounded-xl border border-slate-800 bg-slate-900">
//           <div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-800 p-6">
//             <div>
//               <h2 className="text-xl font-semibold"></h2>
//               <p className="mt-1 text-sm text-slate-400">
//                 {selectedRun
//                   ? `Run: ${selectedRun}`
//                   : "Select a training run to inspect its output."}
//               </p>
//             </div>

//             {selectedRun && (
//               <button
//                 onClick={() => void loadLogs(selectedRun)}
//                 disabled={logsLoading}
//                 className="rounded-lg border border-slate-700 px-4 py-2 text-sm hover:bg-slate-800 disabled:opacity-50"
//               >
//                 {logsLoading ? "Loading..." : "Reload logs"}
//               </button>
//             )}
//           </div>

//           <div className="min-h-56 bg-black p-5">
//             {logsLoading ? (
//               <p className="text-sm text-slate-400">Loading logs...</p>
//             ) : logsError ? (
//               <pre className="whitespace-pre-wrap wrap-break-wordword text-sm text-red-400">
//                 {logsError}
//               </pre>
//             ) : logs ? (
//               <pre className="max-h-128 overflow-auto whitespace-pre-wrap wrap-break-word font-mono text-sm leading-6 text-emerald-300">
//                 {logs}
//               </pre>
//             ) : (
//               <p className="text-sm text-slate-500">
//                 Training output will appear here when you select a run.
//               </p>
//             )}
//           </div>
//         </section> */}
//   </div>
// );
