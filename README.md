# Open-source Readium LCP server in Go
[![CI](https://github.com/amirHdev/ebook-lcp-server/actions/workflows/go.yml/badge.svg)](https://github.com/amirHdev/ebook-lcp-server/actions/workflows/ci.yml)
[![CodeQL](https://github.com/amirHdev/ebook-lcp-server/actions/workflows/codeql.yml/badge.svg)](https://github.com/amirHdev/ebook-lcp-server/actions/workflows/codeql.yml)
[![Release](https://img.shields.io/github/v/release/amirHdev/ebook-lcp-server)](https://github.com/amirHdev/ebook-lcp-server/releases)
[![License](https://img.shields.io/github/license/amirHdev/ebook-lcp-server)](https://github.com/amirHdev/ebook-lcp-server/blob/main/LICENSE)
[![Docker image](https://img.shields.io/badge/docker-ghcr.io%2Famirhdev%2Febook--lcp--server-blue?logo=docker)](https://github.com/amirHdev/ebook-lcp-server/pkgs/container/ebook-lcp-server)

Protect EPUB and PDF files with Readium LCP DRM.

- Encrypt books
- Generate licenses
- Self-hosted
- REST API
- Docker ready
- PostgreSQL support
- Works with Thorium Reader

```bash
docker pull ghcr.io/amirhdev/ebook-lcp-server:latest
```

```bash
docker compose up --build
```

```bash
sh scripts/demo-local.sh
```

## What is in the repo

The project is centered on the normal LCP workflow: ingest a book, protect it, create licenses for readers, and expose enough API around that flow to run it as a service.

- REST and GraphQL APIs for publications, licenses, status, and admin tasks
- JWT auth with admin and publisher roles
- PostgreSQL-backed repositories
- Readium `lcpencrypt` integration for EPUB and PDF processing
- Admin UI
- OpenAPI files and docs endpoints
- Audit log endpoint
- Webhook delivery
- Per-tenant publication and license scoping
- Docker, Kubernetes, K3s, ArgoCD, Prometheus, and Grafana files
- Public-domain EPUB example under `examples/pride-and-prejudice`

It is still moving, but the main pieces are here already.

## Local stack

For local work, the easiest path is the Compose stack. It gives you the API, the Readium sidecars, storage, and the small admin surface in one place.

```bash
docker compose up --build
```

Services started by the current compose file:

| Service | URL |
| --- | --- |
| API | `http://localhost:8080` |
| Admin UI | `http://localhost:5173` |
| PostgreSQL | `localhost:5432` |
| Readium LCP server | `http://localhost:8989` |
| Readium LSD server | `http://localhost:8990` |
| Swagger UI | `http://localhost:8081` |
| MinIO API | `http://localhost:9000` |
| MinIO console | `http://localhost:9001` |
| Prometheus | `http://localhost:9090` |
| Grafana | `http://localhost:3000` |

The admin UI uses these local credentials:

```text
username: admin
password: admin
2FA code: 123456
```

The compose file is meant to bring up the whole local stack, including the Readium LCP and LSD services. It also includes MinIO for the S3 storage path.

Run the local demo after the containers are up:

```bash
sh scripts/demo-local.sh
```

It uploads the sample EPUB, creates a license, and prints the publication ID, license ID, and license URL.

## API examples

The API is token-based. The built-in login route is mainly useful for local work and demos; in a real deployment you will likely put stronger identity management around it.

Get a token:

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin","twoFactor":"123456"}'
```

Check service status:

```bash
curl http://localhost:8080/api/v1/lcp/status \
  -H "Authorization: Bearer $TOKEN"
```

Upload the sample EPUB and create a license:

```bash
sh scripts/demo-local.sh
```

Docs endpoints:

- `http://localhost:8080/swagger.yaml`
- `http://localhost:8080/swagger.json`
- `http://localhost:8080/docs/openapi.yaml`
- `http://localhost:8080/docs/swagger.json`

Public API docs: `https://amirhdev.github.io/ebook-lcp-server/`

Postman collection: `docs/postman/lcp-server.postman_collection.json`

## Example book

There is a sample book in the repo so you do not need to hunt for test content before trying the flow.

The repo includes a public-domain copy of *Pride and Prejudice*:

```bash
examples/pride-and-prejudice/pride-and-prejudice.epub
```

See `examples/pride-and-prejudice/README.md` for the demo notes.

## Reader compatibility

The flow is based on Readium LCP and is intended for LCP-compatible readers such as Thorium Reader. See `docs/reader-compatibility.md` for the current compatibility matrix and the remaining demo work. Fixtures used while checking compatibility live under `examples/lcp-fixtures`.

## Compared with `readium/readium-lcp-server`

The official Readium server is the reference implementation. This project takes a more batteries-included path around the same LCP ecosystem.

| Area | `ebook-lcp-server` | `readium/readium-lcp-server` |
| --- | --- | --- |
| Main focus | Self-hosted API service with a ready local stack | Reference LCP server components |
| Local start | One Compose stack for API, PostgreSQL, MinIO, Swagger UI, and Readium sidecars | Install and run separate Go binaries |
| API surface | REST and GraphQL around publications, licenses, admin tasks, metrics, and docs | License server and status server APIs |
| Storage path | Filesystem or S3-compatible storage, including signed URLs | Filesystem or S3 through `lcpencrypt` |
| Database path | PostgreSQL-first in the local stack | SQLite, MySQL, SQL Server, or PostgreSQL |
| Multi-tenant behavior | Tenant scoping, tenant API keys, tenant webhooks, storage prefixes, and rate limits | Not presented as a built-in tenant layer in the upstream README |
| Developer extras | README demo, Compose stack, Postman collection, hosted docs, audit logs, webhooks, metrics | Upstream CLI/server components and test frontend |

Use the official project when you want the upstream reference pieces directly. Use this repo when you want a more packaged self-hosted service around them.

## Configuration

The defaults are aimed at local development. Start from the example file, then change the values that describe your own deployment.

Start from `.env.example`:

```bash
cp .env.example .env
```

The main settings are:

| Variable | Purpose |
| --- | --- |
| `DB_DSN` | PostgreSQL connection string |
| `LCP_PROVIDER_URI` | Public provider URI written into licenses |
| `LCP_CORE_URL` | Readium LCP core service URL |
| `LCP_CORE_USER`, `LCP_CORE_PASSWORD` | Credentials for the Readium core service |
| `LCP_STORAGE_FS_DIR` | Directory used for encrypted publication files |
| `LCP_S3_ENDPOINT` | S3-compatible endpoint, such as `localhost:9000` for MinIO |
| `LCP_S3_PUBLIC_ENDPOINT` | Public endpoint used when generating signed download URLs |
| `LCP_S3_REGION` | S3 region |
| `LCP_S3_BUCKET` | Bucket for encrypted publication files |
| `LCP_S3_ACCESS_KEY`, `LCP_S3_SECRET_KEY` | S3 credentials |
| `LCP_S3_USE_SSL` | Whether the S3 endpoint uses TLS |
| `LCP_S3_SIGNED_URL_TTL_SECONDS` | Lifetime for signed download URLs |
| `JWT_SECRET` | Secret used to sign API tokens |
| `ADMIN_USERNAME`, `ADMIN_PASSWORD` | Admin login |
| `PUBLISHER_USERNAME`, `PUBLISHER_PASSWORD` | Publisher login |
| `DEFAULT_TENANT_ID` | Tenant ID included in locally issued login tokens |
| `PUBLIC_BASE_URL` | Public base URL used in generated links |
| `STATUS_BASE_URL` | License status server base URL |
| `RATE_LIMIT_RPM` | Per-subject request limit for protected routes |
| `WEBHOOK_URLS` | Comma-separated webhook targets |
| `WEBHOOK_SECRET` | Optional secret used to sign webhook payloads |

Set `LCP_STORAGE_MODE=s3` to store encrypted publication files in S3-compatible storage such as MinIO. When S3 mode is enabled, `/publications/{id}/content` redirects to a short-lived signed URL instead of streaming the object through the API server. The default mode is still `fs`.

The service also has a few integration features that are useful once more than one system is involved:

Webhook events:

- `publication.uploaded`
- `license.created`
- `license.revoked`

Publications and licenses carry a `tenantId`. Tokens issued by the built-in login flow include `DEFAULT_TENANT_ID`, and reads are scoped to that tenant.

Admin audit entries are available at `GET /api/v1/admin/audit`.

## Development

If you prefer to run parts of the stack yourself while working on the code, the API and frontend can still be started separately.

Run the API directly:

```bash
cp .env.example .env
export $(grep -v '^#' .env | xargs)
go run ./cmd/server
```

Run the frontend:

```bash
cd frontend
npm ci
npm run dev
```

Run tests:

```bash
go test ./...
```

## Deployment

The repo contains both a plain Docker path and Kubernetes manifests. The Kubernetes side is more complete today, especially around the Readium core services.

Docker:

```bash
docker build -t lcp-server:local .
docker run -p 8080:8080 --env-file .env lcp-server:local
```

Published image:

```bash
docker pull ghcr.io/amirhdev/ebook-lcp-server:latest
```

Kubernetes:

```bash
kubectl apply -k deploy/overlays/prod
```

Hosted deployment guides:

- `docs/deploy.md`
- `docs/deploy-flyio.md`
- `docs/deploy-railway.md`

K3s scripts, ArgoCD files, monitoring manifests, and production notes are also in the repo.

## Roadmap

The short version is below. The full file is more useful if you want to see the order of work and release gates.

- Hosted OpenAPI docs
- One-click deployment examples
- Reader demos for Thorium, Readium Swift, and Android

The fuller implementation plan is in `docs/roadmap.md`.

## Repository layout

- `cmd/server`: HTTP server wiring
- `internal/adapter/rest`: REST handlers
- `internal/adapter/graphql`: GraphQL schema and resolvers
- `internal/usecase/lcp`: publication and license logic
- `internal/adapter/repository/lcp`: in-memory and PostgreSQL repositories
- `frontend`: admin UI
- `docs`: API, architecture, deployment, and operations docs
- `deploy`: Kubernetes, K3s, monitoring, registry, and ArgoCD manifests
- `examples`: sample books and LCP fixtures

## More docs

- `docs/architecture.md`
- `docs/deployment-guide.md`
- `docs/PRODUCTION.md`
- `docs/security-policy.md`
- `docs/user-manual.md`
- `docs/contract-matrix.md`
- `docs/openapi-rest.yaml`
- `docs/roadmap.md`
