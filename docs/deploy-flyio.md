# Fly.io deployment

This repository already includes a `Dockerfile`, so the Fly path is short:

```bash
fly launch --no-deploy
cp deploy/fly/fly.toml.example fly.toml
fly secrets set \
  DB_DSN='postgres://...' \
  JWT_SECRET='...' \
  ADMIN_USERNAME='admin' \
  ADMIN_PASSWORD='...' \
  ADMIN_2FA_CODE='...' \
  LCP_PROVIDER_URI='https://your-app.fly.dev' \
  PUBLIC_BASE_URL='https://your-app.fly.dev'
fly deploy
```

Before the first deploy, replace `app = "replace-me"` in `fly.toml`. The example file already points Fly traffic at internal port `8080`.

Use an external PostgreSQL database and S3-compatible object storage in production. The single app image is enough for the API, but the full Readium core stack still needs its own deployment plan if you want Fly.io to host every service.
