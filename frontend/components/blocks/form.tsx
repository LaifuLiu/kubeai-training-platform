"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { Controller, useForm } from "react-hook-form";
import { toast } from "@/components/ui/toast";
import * as z from "zod";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

import { useMutateTrainingRuns } from "@/hooks/mutation-training-runs";

export const QUEUES = {
  TRAINING: "ai-training",
  RESEARCH: "ai-research",
  PRIORITY: "ai-priority",
} as const;
export const PRIORITY = {
  NORMAL: "normal",
  HIGH: "high",
};

const QueueItems: {
  label: string;
  value: (typeof QUEUES)[keyof typeof QUEUES];
}[] = [
  { label: "ai-training", value: QUEUES.TRAINING },
  { label: "ai-research", value: QUEUES.RESEARCH },
  { label: "ai-priority", value: QUEUES.PRIORITY },
];

const PriorityItems: {
  label: string;
  value: (typeof PRIORITY)[keyof typeof PRIORITY];
}[] = [
  { label: "kubeai-normal", value: PRIORITY.NORMAL },
  { label: "kubeai-high", value: PRIORITY.HIGH },
];

const ReplicasItems = (() => {
  let items: { label: string; value: number }[] = [];

  for (let i = 1; i <= 8; i++) {
    items.push({ label: String(i), value: i });
  }

  return items;
})();

const MinAvailableItems = (() => {
  let items: { label: string; value: number }[] = [];

  for (let i = 1; i <= 8; i++) {
    items.push({ label: String(i), value: i });
  }

  return items;
})();

const TrainingRunRequestSchema = z.object({
  name: z
    .string()
    .min(5, "name must be at least 5 characters.")
    .max(32, "name must be at most 32 characters."),
  epochs: z.coerce.number<number>().min(1).max(1000),
  queue: z.enum(QUEUES),
  priority: z.enum(PRIORITY),
  replicas: z.number().int().gte(1).lte(8),
  minAvailable: z.number().int().gte(1),
  cpu: z.string(),
  memory: z.string(),
  gpus: z.coerce.number<number>().int().gte(0),
});

export type TrainingRunRequest = z.infer<typeof TrainingRunRequestSchema>;

export function BugReportForm() {
  const form = useForm<TrainingRunRequest>({
    resolver: zodResolver(TrainingRunRequestSchema),
    defaultValues: {
      name: "",
      epochs: 1,
      queue: QUEUES.TRAINING,
      priority: PRIORITY.NORMAL,
      replicas: 1,
      minAvailable: 1,
      cpu: "",
      memory: "",
      gpus: 0,
    },
  });

  const mutateTrainingRuns = useMutateTrainingRuns();

  function onSubmit(data: TrainingRunRequest) {
    mutateTrainingRuns.mutate(data, {
      onSuccess: () => {
        toast.add({
          type: "success",
          description:
            "successfully submitted a new training run, go to training jobs to check out!",
        });
      },
      onError: (error: any) => {
        toast.add({
          type: "error",
          description: `Failed to submit training run: ${error.message}`,
        });
      },
    });
  }

  return (
    <Card className="w-full">
      <CardHeader className="border-b">
        <CardTitle>Create training run</CardTitle>
        <CardDescription>
          Create a new training run by submitting form
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form
          id="form-create-training-run"
          onSubmit={form.handleSubmit(onSubmit)}
        >
          <FieldGroup>
            <div className="grid grid-cols-3 border-b pb-4">
              <Controller
                name="name"
                control={form.control}
                render={({ field, fieldState }) => (
                  <Field data-invalid={fieldState.invalid}>
                    <FieldLabel htmlFor="form-rhf-demo-title">Name</FieldLabel>
                    <Input
                      {...field}
                      id="form-rhf-demo-title"
                      aria-invalid={fieldState.invalid}
                      placeholder="training name"
                      autoComplete="off"
                    />
                    {fieldState.invalid && (
                      <FieldError errors={[fieldState.error]} />
                    )}
                    <FieldDescription>training name</FieldDescription>
                  </Field>
                )}
              />
            </div>

            <div className="grid grid-cols-3 border-b pb-4">
              <Controller
                name="epochs"
                control={form.control}
                render={({ field, fieldState }) => (
                  <Field data-invalid={fieldState.invalid}>
                    <FieldLabel htmlFor="form-rhf-demo-title">
                      Epochs
                    </FieldLabel>
                    <Input
                      {...field}
                      type="number"
                      id="form-rhf-demo-title"
                      aria-invalid={fieldState.invalid}
                      placeholder="input epochs"
                      autoComplete="off"
                    />
                    {fieldState.invalid && (
                      <FieldError errors={[fieldState.error]} />
                    )}
                    <FieldDescription>epochs to training</FieldDescription>
                  </Field>
                )}
              />
            </div>

            <div className="grid grid-cols-3 border-b pb-4">
              <Controller
                name="queue"
                control={form.control}
                render={({ field, fieldState }) => (
                  <Field data-invalid={fieldState.invalid}>
                    <FieldLabel htmlFor="form-rhf-demo-description">
                      Queue
                    </FieldLabel>
                    <Select
                      onValueChange={field.onChange}
                      items={QueueItems}
                      value={field.value}
                    >
                      <SelectTrigger className="w-full max-w-48">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectGroup>
                          <SelectLabel>Queue</SelectLabel>
                          {QueueItems.map((item) => (
                            <SelectItem key={item.value} value={item.value}>
                              {item.label}
                            </SelectItem>
                          ))}
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                    <FieldDescription>k8s queue</FieldDescription>
                    {fieldState.invalid && (
                      <FieldError errors={[fieldState.error]} />
                    )}
                  </Field>
                )}
              />

              <Controller
                name="priority"
                control={form.control}
                render={({ field, fieldState }) => (
                  <Field data-invalid={fieldState.invalid}>
                    <FieldLabel htmlFor="form-rhf-demo-description">
                      Priority
                    </FieldLabel>
                    <Select
                      items={PriorityItems}
                      value={field.value}
                      onValueChange={field.onChange}
                    >
                      <SelectTrigger className="w-full max-w-48">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectGroup>
                          <SelectLabel>Priority</SelectLabel>
                          {PriorityItems.map((item) => (
                            <SelectItem key={item.value} value={item.value}>
                              {item.label}
                            </SelectItem>
                          ))}
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                    <FieldDescription>k8s priority</FieldDescription>
                    {fieldState.invalid && (
                      <FieldError errors={[fieldState.error]} />
                    )}
                  </Field>
                )}
              />
            </div>

            <div className="grid grid-cols-3 border-b pb-4">
              <Controller
                name="replicas"
                control={form.control}
                render={({ field, fieldState }) => (
                  <Field data-invalid={fieldState.invalid}>
                    <FieldLabel htmlFor="form-rhf-demo-description">
                      Replicas
                    </FieldLabel>
                    <Select
                      items={ReplicasItems}
                      value={field.value}
                      onValueChange={field.onChange}
                    >
                      <SelectTrigger className="w-full max-w-48">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectGroup>
                          <SelectLabel>Replicas</SelectLabel>
                          {ReplicasItems.map((item) => (
                            <SelectItem key={item.value} value={item.value}>
                              {item.label}
                            </SelectItem>
                          ))}
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                    <FieldDescription>
                      World side & k8s volcano job replicas
                    </FieldDescription>
                    {fieldState.invalid && (
                      <FieldError errors={[fieldState.error]} />
                    )}
                  </Field>
                )}
              />

              <Controller
                name="minAvailable"
                control={form.control}
                render={({ field, fieldState }) => (
                  <Field data-invalid={fieldState.invalid}>
                    <FieldLabel htmlFor="form-rhf-demo-description">
                      minAvailable
                    </FieldLabel>
                    <Select
                      items={MinAvailableItems}
                      value={field.value}
                      onValueChange={field.onChange}
                    >
                      <SelectTrigger className="w-full max-w-48">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectGroup>
                          <SelectLabel>minAvailable</SelectLabel>
                          {MinAvailableItems.map((item) => (
                            <SelectItem key={item.value} value={item.value}>
                              {item.label}
                            </SelectItem>
                          ))}
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                    <FieldDescription>
                      k8s volcano job minAvailable
                    </FieldDescription>
                    {fieldState.invalid && (
                      <FieldError errors={[fieldState.error]} />
                    )}
                  </Field>
                )}
              />
            </div>

            <div className="grid grid-cols-3 gap-4">
              <Controller
                name="cpu"
                control={form.control}
                render={({ field, fieldState }) => (
                  <Field data-invalid={fieldState.invalid}>
                    <FieldLabel htmlFor="form-rhf-demo-description">
                      Cpu
                    </FieldLabel>
                    <Input
                      {...field}
                      id="form-rhf-demo-title"
                      aria-invalid={fieldState.invalid}
                      placeholder="500m"
                      autoComplete="off"
                    />
                    <FieldDescription>cpu</FieldDescription>
                    {fieldState.invalid && (
                      <FieldError errors={[fieldState.error]} />
                    )}
                  </Field>
                )}
              />

              <Controller
                name="memory"
                control={form.control}
                render={({ field, fieldState }) => (
                  <Field data-invalid={fieldState.invalid}>
                    <FieldLabel htmlFor="form-rhf-demo-description">
                      Memory
                    </FieldLabel>
                    <Input
                      {...field}
                      id="form-rhf-demo-title"
                      aria-invalid={fieldState.invalid}
                      placeholder="1Gi"
                      autoComplete="off"
                    />

                    <FieldDescription>memory</FieldDescription>
                    {fieldState.invalid && (
                      <FieldError errors={[fieldState.error]} />
                    )}
                  </Field>
                )}
              />

              <Controller
                name="gpus"
                control={form.control}
                render={({ field, fieldState }) => (
                  <Field data-invalid={fieldState.invalid}>
                    <FieldLabel htmlFor="form-rhf-demo-description">
                      Gpus
                    </FieldLabel>
                    <Input
                      {...field}
                      type="number"
                      id="form-rhf-demo-title"
                      aria-invalid={fieldState.invalid}
                      placeholder="input gpus"
                      autoComplete="off"
                    />
                    {fieldState.invalid && (
                      <FieldError errors={[fieldState.error]} />
                    )}
                    <FieldDescription>gpus</FieldDescription>
                  </Field>
                )}
              />
            </div>
          </FieldGroup>
        </form>
      </CardContent>
      <CardFooter>
        <Field orientation="horizontal" className="justify-between">
          <Button
            type="submit"
            form="form-create-training-run"
            disabled={mutateTrainingRuns.isPending}
          >
            Start training
          </Button>
          <Button type="button" variant="outline" onClick={() => form.reset()}>
            Reset
          </Button>
        </Field>
      </CardFooter>
    </Card>
  );
}
