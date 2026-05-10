# LCP Server User Manual

## Sign In

`POST /api/v1/auth/login`

Body:

```json
{
  "username": "amiradmin",
  "password": "<admin password>",
  "twoFactor": "<2fa code>"
}
```

Response:

```json
{
  "token": "<jwt>",
  "role": "admin",
  "subject": "amiradmin",
  "expiresAt": "2026-05-16T00:00:00Z"
}
```

The same endpoint also accepts the publisher account and returns `role: "publisher"`.

Use the publisher account for catalog and ingestion workflows. Use the admin account for platform operations and metrics.

## Publication Catalog

`GET /api/v1/publications`

Lists catalog items available to the authenticated user.

`POST /api/v1/publications`

Creates a publication record. The body can include:

- `title`
- `authors`
- `language`
- `subjects`
- `tags`
- `status`
- `encrypted_uri`
- `checksum`
- `license_duration_days`
- `file` as base64 when uploading a new publication

`GET /api/v1/publications/{publicationId}`

Returns one publication record.

`PATCH /api/v1/publications/{publicationId}`

Updates catalog metadata.

`POST /api/v1/publications/{publicationId}/activate`

Marks a publication active.

`POST /api/v1/publications/{publicationId}/deactivate`

Marks a publication inactive.

## Upload And Process

`POST /api/v1/lcp/process`

Headers:

```http
Authorization: Bearer <jwt>
Content-Type: application/json
```

Body:

```json
{
  "title": "Example Book",
  "file": "base64-encoded-content"
}
```

Response:

```json
{
  "id": "process-id",
  "status": "completed",
  "publicationId": "publication-id",
  "createdAt": "2026-05-09T00:00:00Z",
  "updatedAt": "2026-05-09T00:00:00Z"
}
```

The dashboard now provides a file picker. It reads the selected publication file in the browser, base64-encodes it, and sends it to this endpoint.

`/api/v1/lcp/process` does not require `X-2FA-Code`. Admin 2FA is reserved for `/api/v1/admin/*` endpoints such as metrics.

## Check Status

`GET /api/v1/lcp/status?id=<process-id>`

Omit `id` to list known process statuses and service uptime.

## Admin Metrics

`GET /api/v1/admin/metrics`

Headers:

```http
Authorization: Bearer <admin-jwt>
X-2FA-Code: <configured-code>
```

The response includes uptime, process count, and request counters.

## Admin Users

`GET /api/v1/admin/users`

Returns the admin-visible user list.

`POST /api/v1/admin/users/{userId}/verify`

Marks a user verified.

`POST /api/v1/admin/users/{userId}/unverify`

Marks a user unverified.

## API Docs

Swagger/OpenAPI is exposed by the running service at:

- `/swagger.yaml`
- `/swagger.json`
- `/docs/openapi.yaml`
- `/docs/swagger.json`
