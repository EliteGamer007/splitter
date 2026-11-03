package config

import "testing"

func TestFreeTierPoolDefaults(t *testing.T) {
	t.Setenv("DB_MAX_CONNS", "")
	t.Setenv("DB_MIN_CONNS", "")
	cfg := Load()
	if cfg.Database.MaxConns != 4 || cfg.Database.MinConns != 0 {
		t.Fatalf("idle-friendly pool expected max4/min0, got %d/%d", cfg.Database.MaxConns, cfg.Database.MinConns)
	}
}

func TestFederationConfigurationPreservesFeature(t *testing.T) {
	t.Setenv("FEDERATION_ENABLED", "true")
	if !Load().Federation.Enabled {
		t.Fatal("configured federation must be enabled")
	}
	t.Setenv("FEDERATION_ENABLED", "false")
	if Load().Federation.Enabled {
		t.Fatal("explicit federation disable ignored")
	}
}

func TestPoolConfigurationClampsUnsafeValues(t *testing.T) {
	t.Setenv("DB_MAX_CONNS", "1000")
	t.Setenv("DB_MIN_CONNS", "999")
	cfg := Load()
	if cfg.Database.MaxConns > 8 || cfg.Database.MinConns != 0 {
		t.Fatalf("unbounded pool: max%d/min%d", cfg.Database.MaxConns, cfg.Database.MinConns)
	}
}

func TestProductionRequiresExplicitStrongSecret(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv("JWT_SECRET", "")
	if Load().JWT.Secret != "" {
		t.Fatal("production must not supply a known JWT fallback")
	}
}
