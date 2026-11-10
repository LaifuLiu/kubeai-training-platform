"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import Link from "next/link";
import {
  Table,
  TableBody,
  TableCell,
  TableFooter,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useTrainingRuns } from "@/hooks/query-training-runs";
import { Button } from "@/components/ui/button";
import { TrainingRunsPagination } from "@/components/blocks/training-runs-pagination";
import { usePagination } from "@/hooks/use-pagination";

export default function Page() {
  const { page, changePage, size } = usePagination({
    initialPage: 1,
    initialSize: 10,
  });
  const { data, refetch } = useTrainingRuns({
    initialPage: page,
    initialSize: size,
  });

  return (
    <Card>
      <CardHeader className="flex justify-between">
        <div>
          <CardTitle>Training jobs</CardTitle>
          <p className="mt-1 text-sm font-extralight">
            Refreshes automatically every 5 seconds.
          </p>
        </div>
        <div>
          <Button
            onClick={() => {
              refetch();
            }}
          >
            Refresh
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Epochs</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Pods (A/S/F)</TableHead>
              <TableHead>Created</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data?.data?.map((run) => (
              <TableRow key={run.name}>
                <TableCell>
                  <Link href={`/training-jobs/${run.name}`}>{run.name}</Link>
                </TableCell>
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
              </TableRow>
            ))}

            {data?.data?.length === 0 && (
              <TableRow>
                <TableCell colSpan={6}>
                  No training runs found. Create your first run above.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
          <TableFooter className="bg-none">
            <TableRow>
              <TableCell colSpan={6}>
                <TrainingRunsPagination
                  changePage={changePage}
                  page={page}
                  totalPages={data?.pagination.total_pages ?? 0}
                />
              </TableCell>
            </TableRow>
          </TableFooter>
        </Table>
      </CardContent>
    </Card>
  );
}
