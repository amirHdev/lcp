# Operations

This page keeps the boring but important work in one place.

## Backup and restore check

Run the smoke check against the same PostgreSQL database used by the service:

```bash
DB_DSN='postgres://user:pass@localhost:5432/dbname?sslmode=disable' \
  scripts/backup-restore-smoke.sh
```

The script writes a SQL dump and runs a small restore probe inside a transaction. It is not a full disaster-recovery drill, but it catches broken credentials, missing tooling, and a database that cannot be read back.

## Load-test baseline

Use k6 against an authenticated status endpoint:

```bash
TOKEN='...' BASE_URL='http://localhost:8080' k6 run scripts/loadtest-smoke.js
```

The current baseline is intentionally modest: `20` virtual users for `30s`, less than `1%` failed requests, and p95 below `200 ms`. Keep the numbers in version control whenever you change storage, auth, or license generation paths.

## Upgrade

1. Back up PostgreSQL and publication storage.
2. Deploy the new build to staging first.
3. Run `go test ./...`, the backup smoke check, and the k6 smoke test.
4. Deploy the new build to production.
5. Confirm `/readyz`, `/metrics`, publication downloads, and one license creation.

## Rollback

1. Re-deploy the last known good image.
2. Re-run `/readyz` and one license creation.
3. If a migration changed data shape, restore the database backup taken before the upgrade.
4. Record the failed version and the reason before trying again.

## Tenant settings

The admin tenant API can now drive real runtime behavior:

- `apiKeys`: service credentials that become tenant-aware auth claims.
- `webhookUrls`: tenant-specific webhook targets.
- `storagePrefix`: tenant-specific S3 object prefix.
- `rateLimitRpm`: tenant-specific request limit.

Example:

```bash
curl -X POST http://localhost:8080/api/v1/admin/tenants \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-2FA-Code: 123456" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "publisher-a",
    "name": "Publisher A",
    "apiKeys": [{"key": "publisher-a-key", "subject": "publisher-a-service", "role": "publisher"}],
    "webhookUrls": ["https://publisher-a.example/webhooks/lcp"],
    "storagePrefix": "tenants/publisher-a/publications",
    "rateLimitRpm": 300
  }'
```
