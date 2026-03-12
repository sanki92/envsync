package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"
	"github.com/sanki92/envsync/internal/config"
	"github.com/sanki92/envsync/internal/crypto"
	"github.com/sanki92/envsync/internal/team"
	"github.com/sanki92/envsync/internal/vault"
)

func TestAddRemoveTeamWorkflow(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env.local")
	vaultPath := filepath.Join(dir, ".env.vault")
	teamPath := filepath.Join(dir, ".envteam")

	privA, pubA, _ := crypto.GenerateKeypair()
	privB, pubB, _ := crypto.GenerateKeypair()

	tf := team.NewTeamFile("alice", "SHA256:aaa", pubA, "alice")
	team.WriteTeamFile(teamPath, tf)

	envContent := "SECRET_KEY=super-secret\nNEXT_PUBLIC_URL=https://example.com\n"
	os.WriteFile(envPath, []byte(envContent), 0644)

	entries, _ := vault.ReadEnvFile(envPath)
	rA, _ := crypto.ParseRecipient(pubA)
	locked, err := vault.LockVault(entries, []age.Recipient{rA}, []string{"alice"})
	if err != nil {
		t.Fatalf("initial lock: %v", err)
	}
	vault.WriteVaultFile(vaultPath, locked)

	idA, _ := crypto.ParseIdentity(privA)
	unlocked, err := vault.UnlockVault(locked, []age.Identity{idA})
	if err != nil {
		t.Fatalf("alice unlock: %v", err)
	}
	m := vault.EnvMap(unlocked)
	if m["SECRET_KEY"] != "super-secret" {
		t.Fatalf("alice got wrong SECRET_KEY: %q", m["SECRET_KEY"])
	}

	tf, _ = team.ReadTeamFile(teamPath)
	tf.AddMember("bob", "SHA256:bbb", pubB, "alice")
	team.WriteTeamFile(teamPath, tf)

	entries, _ = vault.ReadEnvFile(envPath)
	pubKeys := tf.GetPublicKeys()
	recipients, _ := crypto.ParseRecipients(pubKeys)
	locked, err = vault.LockVault(entries, recipients, tf.MemberNames())
	if err != nil {
		t.Fatalf("re-encrypt after add: %v", err)
	}
	vault.WriteVaultFile(vaultPath, locked)

	idB, _ := crypto.ParseIdentity(privB)
	unlocked, err = vault.UnlockVault(locked, []age.Identity{idB})
	if err != nil {
		t.Fatalf("bob unlock: %v", err)
	}
	m = vault.EnvMap(unlocked)
	if m["SECRET_KEY"] != "super-secret" {
		t.Fatalf("bob got wrong SECRET_KEY: %q", m["SECRET_KEY"])
	}
	if m["NEXT_PUBLIC_URL"] != "https://example.com" {
		t.Fatalf("bob got wrong NEXT_PUBLIC_URL: %q", m["NEXT_PUBLIC_URL"])
	}

	unlocked, err = vault.UnlockVault(locked, []age.Identity{idA})
	if err != nil {
		t.Fatalf("alice unlock after add: %v", err)
	}
	m = vault.EnvMap(unlocked)
	if m["SECRET_KEY"] != "super-secret" {
		t.Fatalf("alice got wrong SECRET_KEY after add: %q", m["SECRET_KEY"])
	}

	tf, _ = team.ReadTeamFile(teamPath)
	tf.RemoveMember("bob")
	team.WriteTeamFile(teamPath, tf)

	pubKeys = tf.GetPublicKeys()
	recipients, _ = crypto.ParseRecipients(pubKeys)
	locked, err = vault.LockVault(entries, recipients, tf.MemberNames())
	if err != nil {
		t.Fatalf("re-encrypt after remove: %v", err)
	}
	vault.WriteVaultFile(vaultPath, locked)

	unlocked, err = vault.UnlockVault(locked, []age.Identity{idA})
	if err != nil {
		t.Fatalf("alice unlock after remove: %v", err)
	}
	m = vault.EnvMap(unlocked)
	if m["SECRET_KEY"] != "super-secret" {
		t.Fatalf("alice got wrong SECRET_KEY after remove: %q", m["SECRET_KEY"])
	}

	_, err = vault.UnlockVault(locked, []age.Identity{idB})
	if err == nil {
		t.Fatal("bob should NOT be able to decrypt after removal")
	}

	tf, _ = team.ReadTeamFile(teamPath)
	if len(tf.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(tf.Members))
	}
	if _, ok := tf.GetMember("alice"); !ok {
		t.Fatal("alice should still be in team")
	}
}

func TestJoinWorkflow(t *testing.T) {
	privKey, pubKey, err := crypto.GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair: %v", err)
	}

	if err := config.SaveKeypair(privKey, pubKey); err != nil {
		t.Fatalf("SaveKeypair: %v", err)
	}

	loaded, err := config.LoadPublicKey()
	if err != nil {
		t.Fatalf("LoadPublicKey: %v", err)
	}
	if strings.TrimSpace(loaded) != pubKey {
		t.Fatalf("public key mismatch: got %q, want %q", loaded, pubKey)
	}

	loadedPriv, err := config.LoadPrivateKey()
	if err != nil {
		t.Fatalf("LoadPrivateKey: %v", err)
	}

	id, err := crypto.ParseIdentity(strings.TrimSpace(loadedPriv))
	if err != nil {
		t.Fatalf("ParseIdentity: %v", err)
	}

	r, _ := crypto.ParseRecipient(pubKey)
	ciphertext, _ := crypto.Encrypt("test-secret", []age.Recipient{r})
	plaintext, err := crypto.Decrypt(ciphertext, []age.Identity{id})
	if err != nil {
		t.Fatalf("decrypt with saved key: %v", err)
	}
	if plaintext != "test-secret" {
		t.Fatalf("roundtrip failed: %q", plaintext)
	}
}

func TestFullTeamLifecycle(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env.local")
	vaultPath := filepath.Join(dir, ".env.vault")
	teamPath := filepath.Join(dir, ".envteam")

	privOwner, pubOwner, _ := crypto.GenerateKeypair()
	privDev1, pubDev1, _ := crypto.GenerateKeypair()
	privDev2, pubDev2, _ := crypto.GenerateKeypair()

	envContent := "DB_URL=postgres://localhost/prod\nAPI_KEY=sk_live_123\nNEXT_PUBLIC_APP=myapp\n"
	os.WriteFile(envPath, []byte(envContent), 0644)

	tf := team.NewTeamFile("owner", "SHA256:owner", pubOwner, "owner")
	team.WriteTeamFile(teamPath, tf)

	entries, _ := vault.ReadEnvFile(envPath)
	rOwner, _ := crypto.ParseRecipient(pubOwner)
	locked, _ := vault.LockVault(entries, []age.Recipient{rOwner}, []string{"owner"})
	vault.WriteVaultFile(vaultPath, locked)

	tf, _ = team.ReadTeamFile(teamPath)
	tf.AddMember("dev1", "SHA256:dev1", pubDev1, "owner")
	team.WriteTeamFile(teamPath, tf)

	pubKeys := tf.GetPublicKeys()
	recipients, _ := crypto.ParseRecipients(pubKeys)
	locked, _ = vault.LockVault(entries, recipients, tf.MemberNames())
	vault.WriteVaultFile(vaultPath, locked)

	tf, _ = team.ReadTeamFile(teamPath)
	tf.AddMember("dev2", "SHA256:dev2", pubDev2, "owner")
	team.WriteTeamFile(teamPath, tf)

	pubKeys = tf.GetPublicKeys()
	recipients, _ = crypto.ParseRecipients(pubKeys)
	locked, _ = vault.LockVault(entries, recipients, tf.MemberNames())
	vault.WriteVaultFile(vaultPath, locked)

	for _, tc := range []struct {
		name string
		priv string
	}{
		{"owner", privOwner},
		{"dev1", privDev1},
		{"dev2", privDev2},
	} {
		id, _ := crypto.ParseIdentity(tc.priv)
		vaultEntries, _ := vault.ReadVaultFile(vaultPath)
		unlocked, err := vault.UnlockVault(vaultEntries, []age.Identity{id})
		if err != nil {
			t.Fatalf("%s cannot decrypt: %v", tc.name, err)
		}
		m := vault.EnvMap(unlocked)
		if m["DB_URL"] != "postgres://localhost/prod" {
			t.Fatalf("%s: wrong DB_URL: %q", tc.name, m["DB_URL"])
		}
		if m["API_KEY"] != "sk_live_123" {
			t.Fatalf("%s: wrong API_KEY: %q", tc.name, m["API_KEY"])
		}
		if m["NEXT_PUBLIC_APP"] != "myapp" {
			t.Fatalf("%s: wrong NEXT_PUBLIC_APP: %q", tc.name, m["NEXT_PUBLIC_APP"])
		}
	}

	tf, _ = team.ReadTeamFile(teamPath)
	tf.RemoveMember("dev1")
	team.WriteTeamFile(teamPath, tf)

	pubKeys = tf.GetPublicKeys()
	recipients, _ = crypto.ParseRecipients(pubKeys)
	locked, _ = vault.LockVault(entries, recipients, tf.MemberNames())
	vault.WriteVaultFile(vaultPath, locked)

	idDev1, _ := crypto.ParseIdentity(privDev1)
	vaultEntries, _ := vault.ReadVaultFile(vaultPath)
	_, err := vault.UnlockVault(vaultEntries, []age.Identity{idDev1})
	if err == nil {
		t.Fatal("dev1 should not be able to decrypt after removal")
	}

	for _, tc := range []struct {
		name string
		priv string
	}{
		{"owner", privOwner},
		{"dev2", privDev2},
	} {
		id, _ := crypto.ParseIdentity(tc.priv)
		vaultEntries, _ := vault.ReadVaultFile(vaultPath)
		unlocked, err := vault.UnlockVault(vaultEntries, []age.Identity{id})
		if err != nil {
			t.Fatalf("%s cannot decrypt after dev1 removal: %v", tc.name, err)
		}
		m := vault.EnvMap(unlocked)
		if m["DB_URL"] != "postgres://localhost/prod" {
			t.Fatalf("%s: wrong DB_URL after removal: %q", tc.name, m["DB_URL"])
		}
	}

	header, err := vault.ReadVaultHeader(vaultPath)
	if err != nil {
		t.Fatalf("ReadVaultHeader: %v", err)
	}
	if len(header.Recipients) != 2 {
		t.Fatalf("expected 2 recipients in header, got %d", len(header.Recipients))
	}
}
