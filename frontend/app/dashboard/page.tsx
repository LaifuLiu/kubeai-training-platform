"use client";

import { BugReportForm } from "@/components/blocks/form";

export default function Page() {
  return (
    <div className="flex">
      <div className="grow-7">
        <BugReportForm />
      </div>

      <div className="grow-3">resources</div>
    </div>
  );
}
