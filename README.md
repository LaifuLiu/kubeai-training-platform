# KubeAI Training Platform

KubeAI Training Platform is a small Kubernetes-based platform for submitting and monitoring machine-learning training runs. It combines a web dashboard, a Go API, and a PyTorch training workload scheduled as a Volcano job.

## Components

- **Frontend** (`frontend/`): Next.js dashboard for submitting training runs, viewing run status and logs, and requesting cancellation.
- **Backend** (`backend/`): Go HTTP API that stores run metadata in PostgreSQL and manages Volcano jobs through the Kubernetes API.
- **Training workload** (`training/`): PyTorch script that trains a linear model on generated sample data and emits JSON progress logs.
- **Kubernetes manifests** (`deploy/`): Namespace, sample job, Volcano queue and priority classes, and frontend/backend resources.

## Requirements

- Node.js 22 and npm
- Go 1.27.1 or later
- Python 3.11 for the training container
- PostgreSQL
- A Kubernetes cluster accessible to the backend, with the Volcano scheduler and Volcano job CRDs installed

The backend uses in-cluster credentials when deployed in Kubernetes. When run locally, it reads the current kubeconfig, normally `~/.kube/config`.

## Run locally

### Frontend

```bash
cd frontend
npm ci
npm run dev
```

Open [http://localhost:3000](http://localhost:3000). The frontend proxies `/backend-api/*` to `BACKEND_API_URL`; this defaults to `http://localhost:8080`.

### Backend

The backend requires a reachable PostgreSQL database and Kubernetes cluster. From `backend/`, set these environment variables (or place them in a local `.env` file):

| Variable | Description |
| --- | --- |
| `DB_HOST` | PostgreSQL hostname |
| `DB_PORT` | PostgreSQL port |
| `DB_NAME` | Database name |
| `DB_USER` | Database user |
| `DB_PASSWORD` | Database password; required |

Then start the API:

```bash
cd backend
go run .
```

The API listens on port `8080` and initializes its `training_runs` schema at startup. Use credentials appropriate for your environment; do not commit `.env` files or secrets.

## API

All application endpoints are under `/api/v1/training-runs`.

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/api/v1/training-runs` | Submit a training run |
| `GET` | `/api/v1/training-runs` | List runs |
| `GET` | `/api/v1/training-runs/{name}` | Get a run |
| `GET` | `/api/v1/training-runs/{name}/logs` | Read run logs |
| `DELETE` | `/api/v1/training-runs/{name}` | Request run deletion |
| `GET` | `/metrics` | Prometheus metrics |

Create-run requests accept JSON fields `name`, `epochs`, `cpu`, `memory`, `priority`, and `replicas`. The name and a positive epoch count are required. Defaults are `500m` CPU, `1Gi` memory, `normal` priority, and one replica; replicas must be between 1 and 8, and priority must be `normal` or `high`.

Example:

```bash
curl -X POST http://localhost:8080/api/v1/training-runs \
  -H 'Content-Type: application/json' \
  -d '{"name":"demo-run","epochs":100,"cpu":"500m","memory":"1Gi","priority":"normal","replicas":1}'
```

## Training image

Build the training container from the repository root:

```bash
docker build -t kubeai-trainer:dev ./training
```

The workload accepts `EPOCHS`, `SEED`, and `LEARNING_RATE` environment variables. Its output is newline-delimited JSON progress events.

Before submitting runs, make sure the image reference used by the backend is available to all cluster nodes. The current reference is configured in `backend/service/server.go`; update it for your registry and image tag when deploying to a different environment.

## Kubernetes deployment

The manifests under `deploy/platform/` deploy the frontend and backend into the `kubeai-training` namespace. Before applying them:

1. Install Volcano and its job CRDs on the cluster.
2. Create the `kubeai-training` namespace (for example, with `kubectl apply -f deploy/base/namespace.yaml`).
3. Create the `kubeai-backend-db` Secret in that namespace with `DB_USER` and `DB_PASSWORD` keys.
4. Configure the PostgreSQL service referenced by the backend manifest, and ensure the training image is available to the cluster.
5. Review the image tags and backend configuration in `deploy/platform/` for your environment.

Apply the Volcano queue and priority classes, then deploy the platform:

```bash
kubectl apply -f deploy/volcano/queue.yaml
kubectl apply -f deploy/base/priorityClass.yaml
kubectl apply -k deploy/platform/
```

The frontend and backend Services are `ClusterIP` resources. Expose the frontend through your preferred ingress or port-forward for local access:

```bash
kubectl -n kubeai-training port-forward service/kubeai-frontend 3000:3000
```

`deploy/base/` also contains a sample standalone training Job. `deploy/volcano/` contains a queue manifest and example Volcano Jobs for testing. Apply sample jobs individually only when you intend to run them; review and adapt them before using in a production cluster.

## Development checks

Frontend:

```bash
cd frontend
npm test
npm run lint
npm run build
```

Backend:

```bash
cd backend
go test ./...
go build ./...
```

GitHub Actions workflows for the frontend and backend are in `.github/workflows/`. They test/build the application and publish container images on pushes to `main`.
