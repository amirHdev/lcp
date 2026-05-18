# Deployment

For a local evaluation, use Docker Compose from the root of the repo:

```bash
docker compose up --build
```

For hosted deployments, start with the provider guide that matches where you want to run the API:

- `docs/deploy-flyio.md`
- `docs/deploy-railway.md`

The API service itself is easy to move because the repo already ships with a `Dockerfile`. A production deployment still needs PostgreSQL, S3-compatible storage, and a plan for the Readium LCP/LSD services.
