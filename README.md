# LCP Server - EBook Manager v2

Lightweight License Content Protection (LCP) server that exposes REST and GraphQL APIs for processing encrypted publications and issuing licenses. The repository includes the DevOps assets needed to run the service on self-hosted K3s with Docker, Kubernetes, GitLab CI/CD, GitHub Actions, and ArgoCD.

## Features

- Endpoints for the contract REST API located at `/api/v1/lcp/process`, `/api/v1/lcp/status`, and `/api/v1/admin/metrics`.
- JWT auth with RBAC roles (admin, publisher, user, guest) and 2FA for admin on `/api/v1/admin/*` via `X-2FA-Code`.
- The GraphQL API endpoint available at `/graphql` for publishing and licensing.
- The pluggable encryption library with support for upstream LCP tool `lcpencrypt` for handling real LCP publication operations.
- In-memory repository implementations with optional JSON-based persistence in `DATA_DIR`.
- Endpoint `/publications/{id}/content` where the client can download encrypted assets by following links from licenses.
- Deployment artefacts for Docker, Kubernetes liveness and readiness probes, PVCs, HPA, Network Policy, backup CronJob, and ArgoCD GitOps pipeline.
- Registry manifests of the self-hosted registry images in `deploy/registry`.
- React/TypeScript frontend with an admin UI for processing publications, viewing status, and metrics in `frontend`.
- CI pipelines that run the formatter, vet, tests, Trivy security scanning, builds, and deploys the Docker container image.

## Configurations

Define the below environment variables (check `.env.example & .env.local` for default values):

- `DB_DSN`: Data source name for database connections (required if used by persistent storage adapter).
- `LCP_PROFILE`: Either `basic` or `production`.
- `LCP_CERTIFICATE`/`LCP_PRIVATE_KEY`: Path for DRM certificates/keys.
- `LCP_STORAGE_MODE`: Can be either `fs`(default) or `s3`.
- `LCP_STORAGE_FS_DIR`: The destination directory for storage of encrypted files when `LCP_STORAGE_MODE` is set to `fs`.
- `LCP_S3_REGION`, `LCP_S3_BUCKET`, `LCP_S3_ACCESS_KEY`, `LCP_S3_SECRET_KEY`: Required when using S3-based storage (`LCP_STORAGE_MODE=s3`).
- `JWT_SECRET`: Secret for any JWT-secured APIs created in the future.
- `ADMIN_2FA_CODE`: Code that may be required for admin roles (optional).
- `SERVER_PORT`: Address for listening service (default `:8080`).
- `PUBLIC_BASE_URL`: Public base URL used to construct download URL (default `http://localhost:PORT`).

## Local development

```bash
# Install dependencies and start the API
cp .env.example .env  # then edit values
export $(grep -v '^#' .env | xargs)
go run ./cmd/server
```

The GraphQL playground will be available at `http://localhost:8080/graphql`.

### REST API

All contract endpoints require a Bearer JWT signed with `JWT_SECRET`. Admin calls under `/api/v1/admin/*` also require `X-2FA-Code` when `ADMIN_2FA_CODE` is set.

```bash
curl -X POST http://localhost:8080/api/v1/lcp/process \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{"title":"Example","file":"aGVsbG8="}'

curl http://localhost:8080/api/v1/lcp/status \
  -H "Authorization: Bearer $JWT"

curl http://localhost:8080/api/v1/admin/metrics \
  -H "Authorization: Bearer $ADMIN_JWT" \
  -H "X-2FA-Code: $ADMIN_2FA_CODE"
```

### GraphQL upload notes

The `uploadPublication` mutation expects the `file` argument as a **base64-encoded string**. Example variables:

```json
{
  "title": "My Book",
  "file": "<base64 encoded content>"
}
```

## Docker

```bash
docker build -t lcp-server:local .
docker run -p 8080:8080 --env-file .env lcp-server:local
```

The multi-stage Dockerfile compiles the Go binary and ships a minimal distroless runtime image suitable for production.

For the full local stack:

```bash
docker compose up --build
```

This starts PostgreSQL, Redis, the backend, the admin UI, Prometheus, and Grafana.

## Frontend

```bash
cd frontend
npm ci
npm run dev
```

The admin UI is available at `http://localhost:5173` in development.

## Kubernetes

Apply the manifests with Kustomize:

```bash
kubectl apply -k deploy/overlays/prod
```

K3s-compatible defaults are used in the deployment: ingress traefik, storage local-path, and built-in cluster components. Change the overlay image links and hostnames as required.
Images are supposed to be stored in the registry hosted internally on `registry.testmedical.ir:5000`.

## ArgoCD

The ArgoCD root application at `deploy/argocd/root-application.yaml` continuously syncs the environment apps from this repo:

```bash
kubectl apply -n argocd -f deploy/argocd/root-application.yaml
```

## GitLab CI/CD

`.gitlab-ci.yml` defines four stages:

1. **lint**: runs `gofmt -l` and `go vet`.
2. **test**: executes `go test ./...`.
3. **build**: builds the Docker image and optionally pushes it when registry credentials exist.
4. **deploy**: applies the Kustomize overlays to the cluster (expects `kubectl` credentials in CI variables).

Set `CI_REGISTRY`, `CI_REGISTRY_USER`, `CI_REGISTRY_PASSWORD`, and `KUBECONFIG` (or in-cluster service account variables) in GitLab to enable full automation.

## Repository layout

- `cmd/server`: HTTP server wiring, GraphQL handler, and LCP use cases.
- `internal/auth`: JWT validation, RBAC, and admin 2FA middleware.
- `internal/adapter/rest`: REST endpoints required by the contract.
- `internal/usecase/lcp`: Business logic for publications and licenses.
- `internal/adapter/graphql`: GraphQL schema and resolvers.
- `deploy/k8s`: Production manifests with Kustomize.
- `frontend`: React/TypeScript admin dashboard.
- `deploy/argocd`: GitOps application definition.
- `deploy/registry`: in-cluster image registry.
- `.gitlab-ci.yml`: Pipeline definition for GitLab.

## Documentation

- `docs/deployment-guide.md`
- `docs/security-policy.md`
- `docs/user-manual.md`
- `docs/contract-matrix.md`
- `docs/architecture.md`
- `docs/openapi-rest.yaml`
- Swagger/OpenAPI is also exposed at runtime on `/swagger.yaml` and `/swagger.json`.
- `docs/acceptance-checklist.md`
- `docs/support-and-knowledge-transfer.md`

## Load Test

```bash
k6 run -e BASE_URL=http://localhost:8080 -e JWT="$JWT" tests/k6/lcp-status.js
```
