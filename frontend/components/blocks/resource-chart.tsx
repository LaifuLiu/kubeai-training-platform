"use client";

import { useNodes } from "@/hooks/use-nodes";
import {
  Progress,
  ProgressLabel,
  ProgressValue,
} from "@/components/ui/progress";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

export function ResourceChart() {
  const { data: nodes, isLoading, error } = useNodes();

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error loading nodes</div>;

  return (
    <Card className="w-full">
      <CardHeader>
        <CardTitle>Node Resource Usage</CardTitle>
      </CardHeader>

      <CardContent>
        {nodes?.map((node) => (
          <Card key={node.name} className="mb-4">
            <CardHeader>
              <CardTitle>{node.name}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex justify-between text-sm mb-1">
                <span className="text-[12px]">CPU Available</span>
                <span className="text-[12px]">
                  {node.allocatable.cpu} / {node.capacity.cpu}
                </span>
              </div>
              <Progress
                value={
                  (parseInt(node.allocatable.cpu) /
                    parseInt(node.capacity.cpu)) *
                  100
                }
              />
            </CardContent>
            <CardContent>
              <div className="flex justify-between text-sm mb-1">
                <span className="text-[12px]">Memory Available</span>
                <Tooltip>
                  <TooltipTrigger className="text-[12px]">
                    {formatKiMemory(node.allocatable.memory)} /{" "}
                    {formatKiMemory(node.capacity.memory)}
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>
                      {formatKiMemory(node.allocatable.memory)} /{" "}
                      {formatKiMemory(node.capacity.memory)}
                    </p>
                  </TooltipContent>
                </Tooltip>
              </div>
              <Progress
                value={calculateMemoryPercentage(
                  node.allocatable.memory,
                  node.capacity.memory,
                )}
              />
            </CardContent>
            <CardContent>
              <div className="flex justify-between text-sm mb-1">
                <span className="text-[12px]">GPUs Available</span>
                <span className="text-[12px]">
                  {node.allocatable.gpus} / {node.capacity.gpus}
                </span>
              </div>
              <Progress
                value={
                  (parseInt(node.allocatable.gpus) /
                    parseInt(node.capacity.gpus)) *
                  100
                }
              />
            </CardContent>
          </Card>
        ))}
      </CardContent>
    </Card>
  );
}

function calculateMemoryPercentage(AvaiStr: string, totalStr: string): number {
  // Strip non-numeric characters to get raw numbers
  const used = parseFloat(AvaiStr.replace(/[^\d.]/g, ""));
  const total = parseFloat(totalStr.replace(/[^\d.]/g, ""));

  if (!total || total === 0) return 0;

  // Calculate percentage and round to 1 decimal place (or integer)
  const percentage = (used / total) * 100;
  return Math.min(Math.round(percentage * 10) / 10, 100);
}

export function formatKiMemory(memStr: string): string {
  // Extract the numeric value from "7602220Ki"
  const val = parseFloat(memStr.replace(/[^\d.]/g, ""));
  if (isNaN(val)) return memStr;

  // Conversion thresholds (in Ki)
  // 1 Mi = 1024 Ki
  // 1 Gi = 1,048,576 Ki (1024 * 1024)
  // 1 Ti = 1,073,741,824 Ki (1024 * 1024 * 1024)

  if (val >= 1024 * 1024 * 1024) {
    return `${(val / (1024 * 1024 * 1024)).toFixed(1)}Ti`;
  }
  if (val >= 1024 * 1024) {
    return `${(val / (1024 * 1024)).toFixed(1)}Gi`;
  }
  if (val >= 1024) {
    return `${(val / 1024).toFixed(1)}Mi`;
  }

  return `${val}Ki`;
}
