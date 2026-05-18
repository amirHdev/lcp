# Production Deployment Guide

This document outlines best practices for deploying **ebook-lcp-server** in a production environment.

## 1. Security Hardening

### Transport Security

- Always put a reverse proxy (**Caddy** or **Traefik** recommended) in front.
- Enable Let's Encrypt TLS.
- Force HTTPS redirect.
- Use TLS 1.3 only.

### Authentication

- Enable **2FA** for all admin users.
- Use strong JWT secrets (`JWT_SECRET`).
- Rotate secrets regularly.

### Secrets Management

- Use Docker secrets, Kubernetes Secrets, or Vault.
- Never commit sensitive values.

### Runtime Security

- Non-root containers (already good).
- Set CPU/Memory limits.
- Read-only filesystem where possible.

## 2. Recommended Production Configuration

- **Database**: PostgreSQL (instead of JSON/in-memory)
- **Storage**: S3 or MinIO
- **Reverse Proxy**: Caddy / Traefik
- **Monitoring**: Prometheus + Grafana + Loki

## 3. Pre-Production Checklist

- [ ] TLS + HTTPS configured
- [ ] 2FA enabled for admins
- [ ] Using PostgreSQL + S3
- [ ] `LCP_STORAGE_MODE=s3` configured with a private bucket
- [ ] `LCP_S3_PUBLIC_ENDPOINT` and signed URL TTL configured
- [ ] Webhook receivers use HTTPS and verify `X-LCP-Signature`
- [ ] `RATE_LIMIT_RPM` reviewed for expected traffic
- [ ] `DEFAULT_TENANT_ID` replaced or external identity provider issues tenant-aware tokens
- [ ] Secrets managed externally
- [ ] Monitoring & alerting active
- [ ] Backups configured and tested
- [ ] Vulnerability scanning in CI

## 4. Useful Commands

```bash
# Vulnerability scan
trivy fs .

# Run production stack
docker compose -f docker-compose.prod.yml up -d
```

## 5. Operational checks

For backup verification, load-test smoke runs, and upgrade/rollback steps, see `docs/operations.md`.
