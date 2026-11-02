export interface Nodes {
  name: string;
  ready: boolean;
  unschedulable: boolean;
  capacity: Allocatable;
  allocatable: Allocatable;
}

export interface Allocatable {
  cpu: string;
  memory: string;
  gpus: string;
}
