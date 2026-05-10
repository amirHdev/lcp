# Contract vs Implemented vs Uncertain

This matrix is the strict read against the contract PDF and the two appendices, plus the repo changes made for the publisher flow.

| Area | Contract / Appendix | Implemented | Uncertain / Notes |
| --- | --- | --- | --- |
| LCP process API | `POST /api/v1/lcp/process` | Yes | Admin, publisher, and user roles can use it; admin 2FA is not required on this endpoint. |
| LCP status API | `GET /api/v1/lcp/status` | Yes | Available to admin, publisher, user, and guest roles. |
| Admin metrics | `GET /api/v1/admin/metrics` | Yes | Requires admin JWT plus `X-2FA-Code`. |
| Auth | JWT bearer auth | Yes | Login now supports admin and publisher credentials. |
| Roles | Admin, user | Yes | `publisher` added as a practical third-party role for catalog/ingestion. |
| Publication catalog | In appendix task for managing publications | Yes | REST endpoints added for list/get/create/update/activate/deactivate. |
| Publisher flow | Third-party publisher access | Yes | Implemented as a simple publisher login plus catalog workspace. |
| Publisher signup / verification UI | Not explicit in the appendices | Partial | No dedicated approval workflow yet; this is the main uncertain product gap. |
| User verification page | Not explicit in the appendices | Yes | Implemented as a simple admin-visible user list with verify/unverify actions. |
| GraphQL | Publication/license workflows | Yes | Kept for integrations and future systems like WordPress. |
| License/user lookup | Status server style license lookup | Yes | `/api/v1/licenses/{licenseId}/user` exposed. |
| Content download | Publication content download endpoint | Yes | `/publications/{publicationId}/content` exposed. |
| Swagger / OpenAPI | Update docs and expose API surface | Yes | Served at `/swagger.yaml`, `/swagger.json`, `/docs/openapi.yaml`, `/docs/swagger.json`. |
| Docker | Containerized deployment | Yes | Dockerfile and compose stack are in place. |
| Kubernetes / K3s | Self-hosted cluster deployment | Yes | K3s + ArgoCD are live. |
| Monitoring | Prometheus/Grafana | Yes | Prometheus is auth-protected; Grafana is live. |
| Backups | Backup job / operational support | Yes | Backup CronJob is deployed. |
| CI/CD | Build, test, scan, deploy automation | Yes | Repo includes CI/CD assets and GitOps sync. |
| Load / performance targets | 100 concurrent users / 1000 RPS target | Uncertain | Needs a real client-side load run in the target environment. |
| Failover / DR validation | Not fully spelled out | Uncertain | Needs an explicit production exercise if the client asks for it. |

## Readable summary

The platform is implemented for:

- admin login and metrics
- publisher login and publication catalog management
- user/process/status/content flows
- monitoring, GitOps, and Kubernetes deployment

The only material uncertainty left is whether the client wants a dedicated publisher onboarding/approval workflow beyond simple login, because that is not clearly mandated in the appendices I checked.
