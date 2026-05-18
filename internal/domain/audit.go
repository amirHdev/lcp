package domain

import "time"

type AuditEntry struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenantId,omitempty"`
	Action     string    `json:"action"`
	Actor      string    `json:"actor"`
	Resource   string    `json:"resource"`
	ResourceID string    `json:"resourceId"`
	CreatedAt  time.Time `json:"createdAt"`
}
