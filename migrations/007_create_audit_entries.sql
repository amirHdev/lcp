CREATE TABLE IF NOT EXISTS audit_entries (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id TEXT NOT NULL DEFAULT 'default',
    action TEXT NOT NULL,
    actor TEXT NOT NULL,
    resource TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
