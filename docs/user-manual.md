# LCP Server User Manual

## Sign In

`POST /api/v1/auth/login`

Body:

```json
{
  "username": "toghyani",
  "password": "<admin password>",
  "twoFactor": "<2fa code>"
}
```

Response:

```json
{
  "token": "<jwt>",
  "role": "admin",
  "subject": "toghyani",
  "expiresAt": "2026-05-16T00:00:00Z"
}
```

The same endpoint also accepts the publisher account and returns `role: "publisher"`.

Use the publisher account for catalog and ingestion workflows. Use the admin account for platform operations and metrics.

The admin and publisher catalog forms both support sharing metadata on the publication record:

- `right_print`
- `right_copy`

Use `0` to disable the right. Leave the field empty to inherit the default from the publication when generating a license.
If the passphrase field is left blank in the UI, the platform will generate one automatically for the license form.

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
- `right_print`
- `right_copy`

`GET /api/v1/publications/{publicationId}`

Returns one publication record.

`PATCH /api/v1/publications/{publicationId}`

Updates catalog metadata.

`POST /api/v1/publications/{publicationId}/activate`

Marks a publication active.

`POST /api/v1/publications/{publicationId}/deactivate`

Marks a publication inactive.

## Issue License

Use the `License Issuance` panel in the admin UI, or call the GraphQL mutation directly.

GraphQL example:

```json
{
  "query": "mutation CreateLicense($publicationID: ID!, $userID: ID!, $passphrase: String!, $hint: String!, $rightPrint: Int, $rightCopy: Int) { createLicense(publicationID: $publicationID, userID: $userID, passphrase: $passphrase, hint: $hint, rightPrint: $rightPrint, rightCopy: $rightCopy) { id publicationID publicationURL rightPrint rightCopy } }",
  "variables": {
    "publicationID": "41740870466d8962468078d3effe8f8e",
    "userID": "reader-01",
    "passphrase": "demo-passphrase",
    "hint": "demo",
    "rightPrint": 10,
    "rightCopy": 2000
  }
}
```

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

## Android PDF Reader Test

After publishing a PDF, test it in a reader that supports LCP-encrypted publications, not a plain Android PDF viewer.

Recommended flow:

1. Use the publication download URL from the API or UI.
2. Open the license file or license URL in the LCP-compatible Android reader.
3. Enter the passphrase or follow the hint shown by the license.
4. Confirm that the reader opens the PDF and that copy/print behavior matches the configured rights.

If the reader does not support LCP, it will not open the encrypted file at all.

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
This endpoint requires `Authorization: Bearer <admin-jwt>` and `X-2FA-Code`.

`POST /api/v1/admin/users/{userId}/verify`

Marks a user verified.
This endpoint also requires `X-2FA-Code`.

`POST /api/v1/admin/users/{userId}/unverify`

Marks a user unverified.
This endpoint also requires `X-2FA-Code`.

## API Docs

Swagger/OpenAPI is exposed by the running service at:

- `/swagger.yaml`
- `/swagger.json`
- `/docs/openapi.yaml`
- `/docs/swagger.json`
