"use client";

import { BugReportForm } from "@/components/blocks/form";
import { ResourceChart } from "@/components/blocks/resource-chart";

export default function Page() {
  return (
    <div className="grid grid-cols-4 gap-4">
      <div className="col-span-3">
        <BugReportForm />
      </div>
      <div className="col-span-1">
        <ResourceChart />
      </div>
    </div>
  );
}
