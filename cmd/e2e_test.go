package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"
	"github.com/sanki92/envsync/internal/crypto"
	gitutil "github.com/sanki92/envsync/internal/git"
	"github.com/sanki92/envsync/internal/team"
	"github.com/sanki92/envsync/internal/vault"
)

func TestE2EFullWorkflow(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)

	envPath := filepath.Join(dir, ".env.local")
	vaultPath := filepath.Join(dir, ".env.vault")
	teamPath := filepath.Join(dir, ".envteam")

	// --- Step 1: Owner init ---
	privOwner, pubOwner, _ := crypto.GenerateKeypair()

	envContent := "DATABASE_URL=postgres://user:pass@localhost:5432/mydb\n" +
		"STRIPE_SECRET_KEY=sk_live_abc123\n" +
		"REDIS_URL=redis://localhost:6379\n" +
		"NEXT_PUBLIC_APP_URL=https://myapp.com\n"
	os.WriteFile(envPath, []byte(envContent), 0644)

	tf := team.NewTeamFile("owner", "SHA256:owner_fp", pubOwner, "owner")
	team.WriteTeamFile(teamPath, tf)

	entries, err := vault.ReadEnvFile(envPath)
	if err != nil {
		t.Fatalf("read env: %v", err)
	}

	rOwner, _ := crypto.ParseRecipient(pubOwner)
	locked, err := vault.LockVault(entries, []age.Recipient{rOwner}, []string{"owner"})
	if err != nil {
		t.Fatalf("init lock: %v", err)
	}
	vault.WriteVaultFile(vaultPath, locked)

	// install hook
	gitutil.InstallPostMergeHook(dir)
	if !gitutil.HasPostMergeHook(dir) {
		t.Fatal("hook should be installed")
	}

	// update gitignore
	gitutil.UpdateGitignore(dir, []string{".env.local", ".envsync/"})

	// verify vault
	vaultEntries, _ := vault.ReadVaultFile(vaultPath)
	for _, e := range vaultEntries {
		if e.Key == "NEXT_PUBLIC_APP_URL" && vault.IsEncrypted(e.Value) {
			t.Fatal("NEXT_PUBLIC_APP_URL should NOT be encrypted")
		}
		if e.Key == "DATABASE_URL" && !vault.IsEncrypted(e.Value) {
			t.Fatal("DATABASE_URL should be encrypted")
		}
	}

	// --- Step 2: Owner unlock roundtrip ---
	os.Remove(envPath)

	idOwner, _ := crypto.ParseIdentity(privOwner)
	unlocked, err := vault.UnlockVault(vaultEntries, []age.Identity{idOwner})
	if err != nil {
		t.Fatalf("owner unlock: %v", err)
	}
	vault.WriteEnvFile(envPath, unlocked)

	recovered, _ := vault.ReadEnvFile(envPath)
	m := vault.EnvMap(recovered)
	if m["DATABASE_URL"] != "postgres://user:pass@localhost:5432/mydb" {
		t.Fatalf("DATABASE_URL roundtrip: %q", m["DATABASE_URL"])
	}
	if m["STRIPE_SECRET_KEY"] != "sk_live_abc123" {
		t.Fatalf("STRIPE_SECRET_KEY roundtrip: %q", m["STRIPE_SECRET_KEY"])
	}
	if m["NEXT_PUBLIC_APP_URL"] != "https://myapp.com" {
		t.Fatalf("NEXT_PUBLIC_APP_URL roundtrip: %q", m["NEXT_PUBLIC_APP_URL"])
	}

	// --- Step 3: Add alice ---
	privAlice, pubAlice, _ := crypto.GenerateKeypair()

	tf, _ = team.ReadTeamFile(teamPath)
	tf.AddMember("alice", "SHA256:alice_fp", pubAlice, "owner")
	team.WriteTeamFile(teamPath, tf)

	entries, _ = vault.ReadEnvFile(envPath)
	pubKeys := tf.GetPublicKeys()
	recipients, _ := crypto.ParseRecipients(pubKeys)
	locked, _ = vault.LockVault(entries, recipients, tf.MemberNames())
	vault.WriteVaultFile(vaultPath, locked)

	// --- Step 4: Alice unlocks (simulates join + unlock) ---
	idAlice, _ := crypto.ParseIdentity(privAlice)
	vaultEntries, _ = vault.ReadVaultFile(vaultPath)
	aliceUnlocked, err := vault.UnlockVault(vaultEntries, []age.Identity{idAlice})
	if err != nil {
		t.Fatalf("alice unlock: %v", err)
	}
	aliceMap := vault.EnvMap(aliceUnlocked)

	// both get identical values
	for k, v := range m {
		if aliceMap[k] != v {
			t.Fatalf("alice %s = %q, owner = %q", k, aliceMap[k], v)
		}
	}

	// --- Step 5: Add bob ---
	privBob, pubBob, _ := crypto.GenerateKeypair()

	tf, _ = team.ReadTeamFile(teamPath)
	tf.AddMember("bob", "SHA256:bob_fp", pubBob, "owner")
	team.WriteTeamFile(teamPath, tf)

	entries, _ = vault.ReadEnvFile(envPath)
	pubKeys = tf.GetPublicKeys()
	recipients, _ = crypto.ParseRecipients(pubKeys)
	locked, _ = vault.LockVault(entries, recipients, tf.MemberNames())
	vault.WriteVaultFile(vaultPath, locked)

	// all three can decrypt
	for _, tc := range []struct {
		name string
		priv string
	}{
		{"owner", privOwner},
		{"alice", privAlice},
		{"bob", privBob},
	} {
		id, _ := crypto.ParseIdentity(tc.priv)
		ve, _ := vault.ReadVaultFile(vaultPath)
		un, err := vault.UnlockVault(ve, []age.Identity{id})
		if err != nil {
			t.Fatalf("%s cannot decrypt: %v", tc.name, err)
		}
		um := vault.EnvMap(un)
		if um["DATABASE_URL"] != m["DATABASE_URL"] {
			t.Fatalf("%s: DATABASE_URL mismatch", tc.name)
		}
	}

	// --- Step 6: Remove alice ---
	tf, _ = team.ReadTeamFile(teamPath)
	tf.RemoveMember("alice")
	team.WriteTeamFile(teamPath, tf)

	entries, _ = vault.ReadEnvFile(envPath)
	pubKeys = tf.GetPublicKeys()
	recipients, _ = crypto.ParseRecipients(pubKeys)
	locked, _ = vault.LockVault(entries, recipients, tf.MemberNames())
	vault.WriteVaultFile(vaultPath, locked)

	// alice CANNOT decrypt
	vaultEntries, _ = vault.ReadVaultFile(vaultPath)
	_, err = vault.UnlockVault(vaultEntries, []age.Identity{idAlice})
	if err == nil {
		t.Fatal("alice should NOT decrypt after removal")
	}

	// owner and bob still can
	for _, tc := range []struct {
		name string
		priv string
	}{
		{"owner", privOwner},
		{"bob", privBob},
	} {
		id, _ := crypto.ParseIdentity(tc.priv)
		ve, _ := vault.ReadVaultFile(vaultPath)
		_, err := vault.UnlockVault(ve, []age.Identity{id})
		if err != nil {
			t.Fatalf("%s cannot decrypt after alice removal: %v", tc.name, err)
		}
	}

	// --- Step 7: Vault header check ---
	header, _ := vault.ReadVaultHeader(vaultPath)
	if header.LastUpdated.IsZero() {
		t.Fatal("vault header should have timestamp")
	}
	if len(header.Recipients) != 2 {
		t.Fatalf("expected 2 recipients in header, got %d: %v", len(header.Recipients), header.Recipients)
	}

	// --- Step 8: Edge case — no SSH key ---
	_, _, err = crypto.ReadLocalSSHPublicKey("/nonexistent/path")
	if err == nil {
		t.Fatal("should error on missing SSH key")
	}

	// --- Step 9: Edge case — wrong key cannot decrypt ---
	privWrong, _, _ := crypto.GenerateKeypair()
	idWrong, _ := crypto.ParseIdentity(privWrong)
	vaultEntries, _ = vault.ReadVaultFile(vaultPath)
	_, err = vault.UnlockVault(vaultEntries, []age.Identity{idWrong})
	if err == nil {
		t.Fatal("random key should NOT decrypt vault")
	}

	// --- Step 10: Edge case — corrupt vault ---
	corruptPath := filepath.Join(dir, ".env.vault.corrupt")
	os.WriteFile(corruptPath, []byte("SECRET=enc:notvalidbase64!!!"), 0644)
	corruptEntries, _ := vault.ReadVaultFile(corruptPath)
	_, err = vault.UnlockVault(corruptEntries, []age.Identity{idOwner})
	if err == nil {
		t.Fatal("corrupt vault should fail to decrypt")
	}

	// --- Step 11: Verify .gitignore ---
	gitignoreData, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
	gitignore := string(gitignoreData)
	if !strings.Contains(gitignore, ".env.local") {
		t.Fatal(".gitignore should contain .env.local")
	}
	if !strings.Contains(gitignore, ".envsync/") {
		t.Fatal(".gitignore should contain .envsync/")
	}
}

func TestE2EEmptyEnvFile(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env.local")
	vaultPath := filepath.Join(dir, ".env.vault")

	os.WriteFile(envPath, []byte(""), 0644)

	priv, pub, _ := crypto.GenerateKeypair()
	r, _ := crypto.ParseRecipient(pub)

	entries, _ := vault.ReadEnvFile(envPath)
	locked, err := vault.LockVault(entries, []age.Recipient{r}, []string{"user"})
	if err != nil {
		t.Fatalf("lock empty env: %v", err)
	}

	vault.WriteVaultFile(vaultPath, locked)

	id, _ := crypto.ParseIdentity(priv)
	vaultEntries, _ := vault.ReadVaultFile(vaultPath)
	unlocked, err := vault.UnlockVault(vaultEntries, []age.Identity{id})
	if err != nil {
		t.Fatalf("unlock empty vault: %v", err)
	}

	m := vault.EnvMap(unlocked)
	if len(m) != 0 {
		t.Fatalf("expected 0 keys, got %d", len(m))
	}
}

func TestE2ESpecialCharValues(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env.local")
	vaultPath := filepath.Join(dir, ".env.vault")

	content := "URL=postgres://user:p@ss=w0rd@host:5432/db?ssl=true&timeout=30\n" +
		"JSON_CONFIG={\"key\": \"value\", \"nested\": {\"a\": 1}}\n" +
		"MULTILINE=line1\\nline2\\nline3\n" +
		"EQUALS_IN_VALUE=base64==data==here\n"
	os.WriteFile(envPath, []byte(content), 0644)

	priv, pub, _ := crypto.GenerateKeypair()
	r, _ := crypto.ParseRecipient(pub)

	entries, _ := vault.ReadEnvFile(envPath)
	locked, err := vault.LockVault(entries, []age.Recipient{r}, []string{"user"})
	if err != nil {
		t.Fatalf("lock special chars: %v", err)
	}

	vault.WriteVaultFile(vaultPath, locked)

	id, _ := crypto.ParseIdentity(priv)
	ve, _ := vault.ReadVaultFile(vaultPath)
	unlocked, err := vault.UnlockVault(ve, []age.Identity{id})
	if err != nil {
		t.Fatalf("unlock special chars: %v", err)
	}

	m := vault.EnvMap(unlocked)
	original := vault.EnvMap(entries)
	for k, v := range original {
		if m[k] != v {
			t.Errorf("key %s: got %q, want %q", k, m[k], v)
		}
	}
}
