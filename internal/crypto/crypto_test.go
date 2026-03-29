package crypto_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"filippo.io/age"
	"github.com/sanki92/envsync/internal/crypto"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	privKey, pubKey, err := crypto.GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair: %v", err)
	}

	recipient, err := crypto.ParseRecipient(pubKey)
	if err != nil {
		t.Fatalf("ParseRecipient: %v", err)
	}

	identity, err := crypto.ParseIdentity(privKey)
	if err != nil {
		t.Fatalf("ParseIdentity: %v", err)
	}

	plaintext := "postgres://user:pass@localhost:5432/mydb"
	ciphertext, err := crypto.Encrypt(plaintext, []age.Recipient{recipient})
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if ciphertext == plaintext {
		t.Fatal("ciphertext should differ from plaintext")
	}

	decrypted, err := crypto.Decrypt(ciphertext, []age.Identity{identity})
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if decrypted != plaintext {
		t.Fatalf("roundtrip failed: got %q, want %q", decrypted, plaintext)
	}
}

func TestMultiRecipient(t *testing.T) {
	priv1, pub1, _ := crypto.GenerateKeypair()
	priv2, pub2, _ := crypto.GenerateKeypair()

	r1, _ := crypto.ParseRecipient(pub1)
	r2, _ := crypto.ParseRecipient(pub2)
	id1, _ := crypto.ParseIdentity(priv1)
	id2, _ := crypto.ParseIdentity(priv2)

	plaintext := "sk_live_abc123"
	ciphertext, err := crypto.Encrypt(plaintext, []age.Recipient{r1, r2})
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	d1, err := crypto.Decrypt(ciphertext, []age.Identity{id1})
	if err != nil {
		t.Fatalf("Decrypt with identity 1: %v", err)
	}
	if d1 != plaintext {
		t.Fatalf("identity 1: got %q, want %q", d1, plaintext)
	}

	d2, err := crypto.Decrypt(ciphertext, []age.Identity{id2})
	if err != nil {
		t.Fatalf("Decrypt with identity 2: %v", err)
	}
	if d2 != plaintext {
		t.Fatalf("identity 2: got %q, want %q", d2, plaintext)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	_, pub1, _ := crypto.GenerateKeypair()
	priv2, _, _ := crypto.GenerateKeypair()

	r1, _ := crypto.ParseRecipient(pub1)
	id2, _ := crypto.ParseIdentity(priv2)

	ciphertext, err := crypto.Encrypt("secret", []age.Recipient{r1})
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	_, err = crypto.Decrypt(ciphertext, []age.Identity{id2})
	if err == nil {
		t.Fatal("expected error decrypting with wrong key")
	}
}

func TestEncryptNoRecipients(t *testing.T) {
	_, err := crypto.Encrypt("secret", nil)
	if err == nil {
		t.Fatal("expected error with no recipients")
	}
}

func TestDecryptNoIdentities(t *testing.T) {
	_, err := crypto.Decrypt("garbage", nil)
	if err == nil {
		t.Fatal("expected error with no identities")
	}
}

func TestParseRecipients(t *testing.T) {
	_, pub1, _ := crypto.GenerateKeypair()
	_, pub2, _ := crypto.GenerateKeypair()

	recipients, err := crypto.ParseRecipients([]string{pub1, pub2, "", "  "})
	if err != nil {
		t.Fatalf("ParseRecipients: %v", err)
	}
	if len(recipients) != 2 {
		t.Fatalf("expected 2 recipients, got %d", len(recipients))
	}
}

func TestFingerprintSSHPublicKey(t *testing.T) {
	testKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl test@example.com"

	info, err := crypto.FingerprintSSHPublicKey(testKey)
	if err != nil {
		t.Fatalf("FingerprintSSHPublicKey: %v", err)
	}

	if info.Type != "ssh-ed25519" {
		t.Fatalf("expected type ssh-ed25519, got %s", info.Type)
	}

	if info.Fingerprint == "" {
		t.Fatal("fingerprint should not be empty")
	}

	if info.Comment != "test@example.com" {
		t.Fatalf("expected comment test@example.com, got %s", info.Comment)
	}
}

func TestSSHEncryptDecryptRoundtrip(t *testing.T) {
	dir := t.TempDir()
	keyPath := dir + "/.ssh/id_ed25519"
	pubPath := keyPath + ".pub"

	generateTestSSHKey(t, dir)

	pubBytes, err := os.ReadFile(pubPath)
	if err != nil {
		t.Fatalf("read pub key: %v", err)
	}
	sshPubKey := strings.TrimSpace(string(pubBytes))

	recipient, err := crypto.ParseSSHRecipient(sshPubKey)
	if err != nil {
		t.Fatalf("ParseSSHRecipient: %v", err)
	}

	plaintext := "super-secret-value-123"
	ciphertext, err := crypto.Encrypt(plaintext, []age.Recipient{recipient})
	if err != nil {
		t.Fatalf("Encrypt with SSH recipient: %v", err)
	}

	identity, err := crypto.LoadSSHIdentity(dir)
	if err != nil {
		t.Fatalf("LoadSSHIdentity: %v", err)
	}

	decrypted, err := crypto.Decrypt(ciphertext, []age.Identity{identity})
	if err != nil {
		t.Fatalf("Decrypt with SSH identity: %v", err)
	}

	if decrypted != plaintext {
		t.Fatalf("SSH roundtrip failed: got %q, want %q", decrypted, plaintext)
	}
}

func TestSSHMultiRecipient(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	generateTestSSHKey(t, dir1)
	generateTestSSHKey(t, dir2)

	pub1, _ := os.ReadFile(dir1 + "/.ssh/id_ed25519.pub")
	pub2, _ := os.ReadFile(dir2 + "/.ssh/id_ed25519.pub")

	recipients, err := crypto.ParseSSHRecipients([]string{
		strings.TrimSpace(string(pub1)),
		strings.TrimSpace(string(pub2)),
	})
	if err != nil {
		t.Fatalf("ParseSSHRecipients: %v", err)
	}

	plaintext := "shared-secret"
	ciphertext, err := crypto.Encrypt(plaintext, recipients)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	for i, dir := range []string{dir1, dir2} {
		id, err := crypto.LoadSSHIdentity(dir)
		if err != nil {
			t.Fatalf("LoadSSHIdentity %d: %v", i, err)
		}
		decrypted, err := crypto.Decrypt(ciphertext, []age.Identity{id})
		if err != nil {
			t.Fatalf("Decrypt %d: %v", i, err)
		}
		if decrypted != plaintext {
			t.Fatalf("user %d: got %q, want %q", i, decrypted, plaintext)
		}
	}
}

func TestParseSSHRecipients(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	generateTestSSHKey(t, dir1)
	generateTestSSHKey(t, dir2)

	pub1, _ := os.ReadFile(dir1 + "/.ssh/id_ed25519.pub")
	pub2, _ := os.ReadFile(dir2 + "/.ssh/id_ed25519.pub")

	recipients, err := crypto.ParseSSHRecipients([]string{
		strings.TrimSpace(string(pub1)),
		strings.TrimSpace(string(pub2)),
		"", "  ",
	})
	if err != nil {
		t.Fatalf("ParseSSHRecipients: %v", err)
	}
	if len(recipients) != 2 {
		t.Fatalf("expected 2 recipients, got %d", len(recipients))
	}
}

func generateTestSSHKey(t *testing.T, homeDir string) {
	t.Helper()
	sshDir := homeDir + "/.ssh"
	os.MkdirAll(sshDir, 0700)

	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", sshDir+"/id_ed25519", "-N", "", "-q")
	if err := cmd.Run(); err != nil {
		t.Skipf("ssh-keygen not available: %v", err)
	}
}
