# Railway deployment

Railway can deploy this service directly from the repository because the project already has a `Dockerfile`.

## Dashboard path

1. Create a new Railway project from the GitHub repository.
2. Add a PostgreSQL service.
3. Add the required environment variables from `.env.example`.
4. Generate a public domain for the app service.
5. Set `PUBLIC_BASE_URL` and `LCP_PROVIDER_URI` to that public domain.

CLI users can deploy the same code path with:

```bash
railway up
```

Because this repo includes a `Dockerfile`, Railway will build from it automatically.

For production, keep encrypted publications in S3-compatible storage instead of local disk. Railway is a good fit for the API service; the Readium core services still need to be deployed alongside it or hosted separately.
