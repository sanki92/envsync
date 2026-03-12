package crypto_test

import (
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
