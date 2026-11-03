package testdb

import "testing"

func TestOnlyExplicitDisposableLoopbackDatabaseIsAllowed(t *testing.T) {
	for _, raw := range []string{"", "postgres://user:secret@production.example/splitter_repair_test", "postgres://user:secret@127.0.0.1/production", "postgres://user:secret@localhost/splitter", "https://localhost/splitter_repair_test", "postgres://localhost/splitter_repair_test?host=production.example", "postgres://localhost/splitter_repair_test?service=production", "postgres://localhost/%73plitter_repair_test?sslmode=disable&dbname=production"} {
		if _, err := ParseLocalURL(raw); err == nil {
			t.Errorf("unsafe test database accepted: %q", raw)
		}
	}
	for _, raw := range []string{"postgres://test@127.0.0.1:55432/splitter_repair_auth?sslmode=disable", "postgresql://test@[::1]/splitter_repair_test"} {
		if _, err := ParseLocalURL(raw); err != nil {
			t.Errorf("disposable test database rejected: %v", err)
		}
	}
}
