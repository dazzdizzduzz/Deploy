package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadEnv(t *testing.T) {
	tempDir := t.TempDir()
	envFile := filepath.Join(tempDir, ".env")

	content := `
# Komentar baris
DB_HOST=127.0.0.1
DB_PORT=5433
DB_USER='myuser'
DB_PASSWORD="mypassword"
DB_NAME=test_db
EMPTY_LINE=

# Variabel lain
CUSTOM_VAR=hello_world
`
	if err := os.WriteFile(envFile, []byte(content), 0o600); err != nil {
		t.Fatalf("gagal membuat file temp: %v", err)
	}

	// Pastikan variabel bersih sebelum pengujian
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_PORT", "")
	t.Setenv("DB_USER", "")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("DB_NAME", "")
	t.Setenv("CUSTOM_VAR", "")

	// Setenv variabel yang sudah ada di sistem agar tidak tertimpa
	t.Setenv("ALREADY_SET", "initial_val")

	LoadEnv(envFile)

	if val := os.Getenv("DB_HOST"); val != "127.0.0.1" {
		t.Errorf("DB_HOST = %q, want %q", val, "127.0.0.1")
	}
	if val := os.Getenv("DB_PORT"); val != "5433" {
		t.Errorf("DB_PORT = %q, want %q", val, "5433")
	}
	if val := os.Getenv("DB_USER"); val != "myuser" {
		t.Errorf("DB_USER = %q, want %q", val, "myuser")
	}
	if val := os.Getenv("DB_PASSWORD"); val != "mypassword" {
		t.Errorf("DB_PASSWORD = %q, want %q", val, "mypassword")
	}
	if val := os.Getenv("CUSTOM_VAR"); val != "hello_world" {
		t.Errorf("CUSTOM_VAR = %q, want %q", val, "hello_world")
	}
}

func TestLoadEnvMultilineAndQuotes(t *testing.T) {
	tempDir := t.TempDir()
	envFile := filepath.Join(tempDir, ".env")

	content := `
# Multiline PEM key format
JWT_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----
MIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQC9XKBT55dHTKXv
-----END PRIVATE KEY-----"

# Escaped newline string format
JWT_PUBLIC_KEY="-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A\n-----END PUBLIC KEY-----\n"

# Double quote escaping
ESCAPED_QUOTE="hello \"world\""
`
	if err := os.WriteFile(envFile, []byte(content), 0o600); err != nil {
		t.Fatalf("gagal membuat file temp: %v", err)
	}

	t.Setenv("JWT_PRIVATE_KEY", "")
	t.Setenv("JWT_PUBLIC_KEY", "")
	t.Setenv("ESCAPED_QUOTE", "")

	LoadEnv(envFile)

	privKey := os.Getenv("JWT_PRIVATE_KEY")
	if !strings.Contains(privKey, "-----BEGIN PRIVATE KEY-----") || !strings.Contains(privKey, "-----END PRIVATE KEY-----") {
		t.Errorf("JWT_PRIVATE_KEY did not parse multiline correctly: %q", privKey)
	}
	if !strings.Contains(privKey, "\n") {
		t.Errorf("JWT_PRIVATE_KEY expected newline characters: %q", privKey)
	}

	pubKey := os.Getenv("JWT_PUBLIC_KEY")
	if !strings.Contains(pubKey, "-----BEGIN PUBLIC KEY-----") || !strings.Contains(pubKey, "\n") {
		t.Errorf("JWT_PUBLIC_KEY did not unescape \\n correctly: %q", pubKey)
	}

	escapedQuote := os.Getenv("ESCAPED_QUOTE")
	if escapedQuote != `hello "world"` {
		t.Errorf("ESCAPED_QUOTE = %q, want %q", escapedQuote, `hello "world"`)
	}
}

func TestLoadEnvProductionPriority(t *testing.T) {
	tempDir := t.TempDir()
	prodFile := filepath.Join(tempDir, ".env.production")
	defaultFile := filepath.Join(tempDir, ".env")

	if err := os.WriteFile(prodFile, []byte("APP_ENV_NAME=production\nGIN_MODE=release\n"), 0o600); err != nil {
		t.Fatalf("gagal menulis prod file: %v", err)
	}
	if err := os.WriteFile(defaultFile, []byte("APP_ENV_NAME=development\nGIN_MODE=debug\n"), 0o600); err != nil {
		t.Fatalf("gagal menulis default file: %v", err)
	}

	t.Setenv("APP_ENV_NAME", "")
	t.Setenv("GIN_MODE", "")

	// Test passing files where production is listed first
	LoadEnv(prodFile, defaultFile)

	if val := os.Getenv("APP_ENV_NAME"); val != "production" {
		t.Errorf("APP_ENV_NAME = %q, want %q", val, "production")
	}
	if val := os.Getenv("GIN_MODE"); val != "release" {
		t.Errorf("GIN_MODE = %q, want %q", val, "release")
	}
}

func TestLoadConfig(t *testing.T) {
	t.Setenv("DB_HOST", "db.example.com")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "postgres")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "ontrack")
	t.Setenv("DB_SSLMODE", "require")

	cfg := LoadConfig()

	if cfg.DBHost != "db.example.com" {
		t.Errorf("DBHost = %q, want %q", cfg.DBHost, "db.example.com")
	}
	expectedDSN := "host=db.example.com port=5432 user=postgres password=secret dbname=ontrack sslmode=require"
	if cfg.DSN() != expectedDSN {
		t.Errorf("DSN = %q, want %q", cfg.DSN(), expectedDSN)
	}
}
