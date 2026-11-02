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
