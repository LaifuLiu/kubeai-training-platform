export interface TrainingRuns {
  name: string;
  namespace: string;
  status: Phase;
  epochs: string;
  replicas: number;
  minAvailable: number;
  createdAt: Date;
  active: number;
  succeeded: number;
  failed: number;
  cpu: string;
  memory: string;
  gpus: number;
  scheduledAt?: Date;
  queueWaitSeconds?: number;
  pods?: Pod[];
}

export interface Pod {
  name: string;
  phase: Phase;
  nodeName: NodeName;
  requests: Requests;
}

export enum NodeName {
  Worker1 = "worker1",
  Worker2 = "worker2",
}

export enum Phase {
  Succeeded = "Succeeded",
}

export interface Requests {
  cpu: string;
  memory: string;
  gpus: string;
}
