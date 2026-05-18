# LCP Server Deployment Guide

## Environments

Use a separate Kubernetes namespace per environment on a self-hosted K3s cluster:

- `lcp-dev`: low resource limits, local image tags, test secrets.
- `lcp-staging`: production-like configuration and real TLS.
- `lcp-prod`: production secrets, persistent volumes, monitoring, backups, and alerting.

## Required Configuration

Set these values before exposing the service:

- `JWT_SECRET`: HMAC secret used to validate Bearer JWTs.
- `ADMIN_2FA_CODE`: second factor required for requests under `/api/v1/admin/*`.
- `PUBLIC_BASE_URL`: public HTTPS origin used in generated publication URLs.
- `LCP_PROVIDER_URI`: provider URI embedded in generated LCP licenses.
- `LCP_CORE_URL`: internal base URL of the Readium LCP core service.
- `STATUS_BASE_URL`: public base URL of the Readium status server.
- `DATA_DIR`: persistent metadata directory, normally `/var/lib/lcp/data`.
- `LCP_STORAGE_FS_DIR`: persistent encrypted publication directory.

The Kubernetes overlay stores placeholder values in `deploy/k8s/secret.yaml`. Replace them with cluster-managed secrets before production.
K3s already gives you Traefik and local-path storage, so no separate ingress controller or CSI layer is required for the baseline setup.

## Local

```bash
cp .env.example .env
export $(grep -v '^#' .env | xargs)
go run ./cmd/server
```

## Kubernetes

```bash
kubectl create namespace lcp-prod
kubectl apply -n lcp-prod -k deploy/overlays/prod
```

The deployment exposes:

- `GET /healthz`: liveness probe.
- `GET /readyz`: readiness probe.
- `POST /api/v1/lcp/process`: authenticated processing endpoint.
- `GET /api/v1/lcp/status`: authenticated processing status endpoint.
- `GET /api/v1/admin/metrics`: admin metrics endpoint with 2FA header.

## Acceptance Checks

1. Run `go test ./...`.
2. Run `docker build -t lcp-server:local .`.
3. Run Trivy with `trivy fs --severity CRITICAL --exit-code 1 .`.
4. Apply `deploy/overlays/prod` to a Kubernetes cluster.
5. Confirm HPA exists with `kubectl get hpa`.
6. Confirm probes pass with `kubectl rollout status deploy/lcp-server`.
7. Confirm backups create daily Jobs from `cronjob/lcp-backup`.

## Load Testing

Use k6 or JMeter against `/api/v1/lcp/status` and `/api/v1/lcp/process` with valid JWTs. The contract target is p95 below 200 ms under 100 concurrent users and 1000 RPS in optimized conditions.

For the repeatable smoke version used by this repository, run `scripts/loadtest-smoke.js` as described in `docs/operations.md`.
