package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"splitter/internal/db"
	"splitter/tests/testdb"
	"strings"
	"testing"
	"time"
)

func TestServerStartupPreservesExistingAccountsAndSchema(t *testing.T) {
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		t.Skip("requires explicit disposable TEST_DATABASE_URL")
	}
	u, err := testdb.ParseLocalURL(raw)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	admin, err := pgxpool.New(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	name := fmt.Sprintf("splitter_repair_startup_%d", time.Now().UnixNano())
	quoted := pgx.Identifier{name}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+quoted); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(context.Background(), "DROP DATABASE "+quoted+" WITH (FORCE)")
	u.Path = "/" + name
	pool, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := db.ApplyMigrations(ctx, pool, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO users(username,email,password_hash,instance_domain,did,public_key,encryption_public_key,role) VALUES('admin1','sentinel@example.test','sentinel-hash','original-domain','did:sentinel','signing-sentinel','encryption-sentinel','user')`)
	if err != nil {
		t.Fatal(err)
	}
	var before int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM schema_migrations").Scan(&before); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "server")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.CommandContext(ctx, "go", "build", "-o", bin, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build server: %v %s", err, output)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := fmt.Sprint(listener.Addr().(*net.TCPAddr).Port)
	listener.Close()
	command := exec.Command(bin)
	password := ""
	if u.User != nil {
		password, _ = u.User.Password()
	}
	overrides := map[string]string{"DB_HOST": u.Hostname(), "DB_PORT": u.Port(), "DB_NAME": name, "DB_USER": u.User.Username(), "DB_PASSWORD": password, "DB_SSLMODE": "disable", "PORT": port, "ENV": "test", "ENV_FILE": filepath.Join(t.TempDir(), "absent.env"), "JWT_SECRET": "test-only-long-secret-for-startup-regression", "FEDERATION_ENABLED": "false"}
	command.Env = overrideEnvironment(os.Environ(), overrides)
	var logs bytes.Buffer
	command.Stdout = &logs
	command.Stderr = &logs
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	stopped := false
	stop := func() {
		if !stopped {
			stopped = true
			command.Process.Kill()
			command.Wait()
		}
	}
	defer stop()
	client := &http.Client{Timeout: 300 * time.Millisecond}
	ready := false
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		response, err := client.Get("http://127.0.0.1:" + port + "/health")
		if err == nil {
			response.Body.Close()
			ready = true
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	stop()
	if !ready {
		t.Fatalf("server failed to listen: %s", logs.String())
	}
	var signing, encryption, hash, role, domain string
	if err := pool.QueryRow(ctx, "SELECT public_key,encryption_public_key,password_hash,role,instance_domain FROM users WHERE username='admin1'").Scan(&signing, &encryption, &hash, &role, &domain); err != nil {
		t.Fatal(err)
	}
	if signing != "signing-sentinel" || encryption != "encryption-sentinel" || hash != "sentinel-hash" || role != "user" || domain != "original-domain" {
		t.Fatalf("startup mutated existing account: signing=%q encryption=%q role=%q domain=%q passwordUnchanged=%v", signing, encryption, role, domain, hash == "sentinel-hash")
	}
	var after, users int
	pool.QueryRow(ctx, "SELECT count(*) FROM schema_migrations").Scan(&after)
	pool.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&users)
	if before != after || users != 1 {
		t.Fatalf("startup applied schema markers or created users: migrations %d to %d users=%d", before, after, users)
	}
}
func overrideEnvironment(base []string, overrides map[string]string) []string {
	result := make([]string, 0, len(base)+len(overrides))
	for _, item := range base {
		key, _, _ := strings.Cut(item, "=")
		if _, exists := overrides[key]; !exists {
			result = append(result, item)
		}
	}
	for key, value := range overrides {
		result = append(result, key+"="+value)
	}
	return result
}
