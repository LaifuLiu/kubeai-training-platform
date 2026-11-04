export interface TrainingRuns {
  data: Datum[];
  pagination: Pagination;
}

export interface Datum {
  name: string;
  namespace: string;
  status: string;
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
  nodeName: string;
  requests: Requests;
}

export enum Phase {
  Succeeded = "Succeeded",
}

export interface Requests {
  cpu: string;
  memory: string;
  gpus: string;
}

export interface Pagination {
  page: number;
  size: number;
  total: number;
  total_pages: number;
}
