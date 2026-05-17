package config

import "testing"

func TestLoadConfig(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("ADMIN_USER", "admin")
	t.Setenv("ADMIN_PASS", "pass")
	t.Setenv("PUBLISHER_USER", "publisher")
	t.Setenv("PUBLISHER_PASS", "pass")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected config")
	}
}
