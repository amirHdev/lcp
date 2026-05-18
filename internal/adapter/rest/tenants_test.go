package rest

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminTenantsHandlerCRUD(t *testing.T) {
	store := NewTenantStore(t.TempDir(), "default")
	handler := NewAdminTenantsHandler(store)

	create := httptest.NewRequest(http.MethodPost, "/api/v1/admin/tenants", bytes.NewBufferString(`{"id":"publisher-a","name":"Publisher A","rateLimitRpm":120}`))
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, create)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createRec.Code)
	}

	get := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants/publisher-a", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, get)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getRec.Code)
	}

	remove := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/tenants/publisher-a", nil)
	removeRec := httptest.NewRecorder()
	handler.ServeHTTP(removeRec, remove)
	if removeRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", removeRec.Code)
	}
}
