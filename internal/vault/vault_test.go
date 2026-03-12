package vault_test

import (
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"github.com/sanki92/envsync/internal/crypto"
	"github.com/sanki92/envsync/internal/vault"
)

func TestReadWriteEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env.local")

	original := []vault.EnvEntry{
		{Comment: "# database config"},
		{Key: "DATABASE_URL", Value: "postgres://localhost:5432/mydb"},
		{Key: "STRIPE_KEY", Value: "sk_live_abc123"},
		{Comment: ""},
		{Key: "NEXT_PUBLIC_URL", Value: "https://example.com"},
	}

	if err := vault.WriteEnvFile(path, original); err != nil {
		t.Fatalf("WriteEnvFile: %v", err)
	}

	parsed, err := vault.ReadEnvFile(path)
	if err != nil {
		t.Fatalf("ReadEnvFile: %v", err)
	}

	if len(parsed) != len(original) {
		t.Fatalf("expected %d entries, got %d", len(original), len(parsed))
	}

	for i, e := range parsed {
		if e.Key != original[i].Key || e.Value != original[i].Value {
			t.Errorf("entry %d: got {%q, %q}, want {%q, %q}", i, e.Key, e.Value, original[i].Key, original[i].Value)
		}
	}
}

func TestEnvMap(t *testing.T) {
	entries := []vault.EnvEntry{
		{Comment: "# comment"},
		{Key: "A", Value: "1"},
		{Key: "B", Value: "2"},
	}
	m := vault.EnvMap(entries)
	if m["A"] != "1" || m["B"] != "2" {
		t.Fatalf("unexpected map: %v", m)
	}
	if len(m) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(m))
	}
}

func TestIsPublicKey(t *testing.T) {
	tests := []struct {
		key    string
		public bool
	}{
		{"NEXT_PUBLIC_URL", true},
		{"REACT_APP_NAME", true},
		{"VITE_API_URL", true},
		{"DATABASE_URL", false},
		{"SECRET_KEY", false},
	}
	for _, tt := range tests {
		if got := vault.IsPublicKey(tt.key); got != tt.public {
			t.Errorf("IsPublicKey(%q) = %v, want %v", tt.key, got, tt.public)
		}
	}
}

func TestLockUnlockRoundtrip(t *testing.T) {
	priv, pub, _ := crypto.GenerateKeypair()
	r, _ := crypto.ParseRecipient(pub)
	id, _ := crypto.ParseIdentity(priv)

	entries := []vault.EnvEntry{
		{Key: "SECRET", Value: "my-secret-value"},
		{Key: "NEXT_PUBLIC_URL", Value: "https://example.com"},
	}

	locked, err := vault.LockVault(entries, []age.Recipient{r}, []string{"testuser"})
	if err != nil {
		t.Fatalf("LockVault: %v", err)
	}

	var secretEntry vault.EnvEntry
	var publicEntry vault.EnvEntry
	for _, e := range locked {
		if e.Key == "SECRET" {
			secretEntry = e
		}
		if e.Key == "NEXT_PUBLIC_URL" {
			publicEntry = e
		}
	}

	if !vault.IsEncrypted(secretEntry.Value) {
		t.Fatal("SECRET should be encrypted")
	}
	if vault.IsEncrypted(publicEntry.Value) {
		t.Fatal("NEXT_PUBLIC_URL should NOT be encrypted")
	}
	if publicEntry.Value != "https://example.com" {
		t.Fatalf("public value changed: %q", publicEntry.Value)
	}

	unlocked, err := vault.UnlockVault(locked, []age.Identity{id})
	if err != nil {
		t.Fatalf("UnlockVault: %v", err)
	}

	m := vault.EnvMap(unlocked)
	if m["SECRET"] != "my-secret-value" {
		t.Fatalf("SECRET roundtrip failed: got %q", m["SECRET"])
	}
	if m["NEXT_PUBLIC_URL"] != "https://example.com" {
		t.Fatalf("NEXT_PUBLIC_URL roundtrip failed: got %q", m["NEXT_PUBLIC_URL"])
	}
}

func TestVaultFileRoundtrip(t *testing.T) {
	priv, pub, _ := crypto.GenerateKeypair()
	r, _ := crypto.ParseRecipient(pub)
	id, _ := crypto.ParseIdentity(priv)

	entries := []vault.EnvEntry{
		{Key: "DB_URL", Value: "postgres://localhost/test"},
		{Key: "API_KEY", Value: "sk_test_123"},
	}

	locked, err := vault.LockVault(entries, []age.Recipient{r}, []string{"alice"})
	if err != nil {
		t.Fatalf("LockVault: %v", err)
	}

	dir := t.TempDir()
	vaultPath := filepath.Join(dir, ".env.vault")

	if err := vault.WriteVaultFile(vaultPath, locked); err != nil {
		t.Fatalf("WriteVaultFile: %v", err)
	}

	readBack, err := vault.ReadVaultFile(vaultPath)
	if err != nil {
		t.Fatalf("ReadVaultFile: %v", err)
	}

	unlocked, err := vault.UnlockVault(readBack, []age.Identity{id})
	if err != nil {
		t.Fatalf("UnlockVault: %v", err)
	}

	m := vault.EnvMap(unlocked)
	if m["DB_URL"] != "postgres://localhost/test" {
		t.Fatalf("DB_URL mismatch: %q", m["DB_URL"])
	}
	if m["API_KEY"] != "sk_test_123" {
		t.Fatalf("API_KEY mismatch: %q", m["API_KEY"])
	}
}

func TestReadVaultHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env.vault")

	content := `# envsync vault - safe to commit
# Recipients: alice, bob
# Last updated: 2026-03-12T10:30:00Z

DATABASE_URL=enc:abc123
`
	os.WriteFile(path, []byte(content), 0644)

	header, err := vault.ReadVaultHeader(path)
	if err != nil {
		t.Fatalf("ReadVaultHeader: %v", err)
	}

	if len(header.Recipients) != 2 {
		t.Fatalf("expected 2 recipients, got %d", len(header.Recipients))
	}
	if header.Recipients[0] != "alice" || header.Recipients[1] != "bob" {
		t.Fatalf("unexpected recipients: %v", header.Recipients)
	}
	if header.LastUpdated.IsZero() {
		t.Fatal("LastUpdated should not be zero")
	}
}

func TestParseQuotedValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")

	content := `SINGLE='hello world'
DOUBLE="hello world"
NONE=hello world
`
	os.WriteFile(path, []byte(content), 0644)

	entries, err := vault.ReadEnvFile(path)
	if err != nil {
		t.Fatalf("ReadEnvFile: %v", err)
	}

	m := vault.EnvMap(entries)
	if m["SINGLE"] != "hello world" {
		t.Fatalf("SINGLE: %q", m["SINGLE"])
	}
	if m["DOUBLE"] != "hello world" {
		t.Fatalf("DOUBLE: %q", m["DOUBLE"])
	}
	if m["NONE"] != "hello world" {
		t.Fatalf("NONE: %q", m["NONE"])
	}
}
