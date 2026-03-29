package crypto

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	"filippo.io/age"
	"filippo.io/age/agessh"
	"golang.org/x/crypto/ssh"
)

func Encrypt(plaintext string, recipients []age.Recipient) (string, error) {
	if len(recipients) == 0 {
		return "", fmt.Errorf("no recipients provided")
	}

	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, recipients...)
	if err != nil {
		return "", fmt.Errorf("age encrypt: %w", err)
	}
	if _, err := io.WriteString(w, plaintext); err != nil {
		return "", fmt.Errorf("write plaintext: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("close writer: %w", err)
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func Decrypt(ciphertext string, identities []age.Identity) (string, error) {
	if len(identities) == 0 {
		return "", fmt.Errorf("no identities provided")
	}

	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}

	r, err := age.Decrypt(bytes.NewReader(raw), identities...)
	if err != nil {
		return "", fmt.Errorf("age decrypt: %w", err)
	}

	var out bytes.Buffer
	if _, err := io.Copy(&out, r); err != nil {
		return "", fmt.Errorf("read plaintext: %w", err)
	}

	return out.String(), nil
}

func GenerateKeypair() (string, string, error) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return "", "", fmt.Errorf("generate identity: %w", err)
	}
	return identity.String(), identity.Recipient().String(), nil
}

func ParseRecipient(pubkey string) (age.Recipient, error) {
	r, err := age.ParseX25519Recipient(pubkey)
	if err != nil {
		return nil, fmt.Errorf("parse recipient %q: %w", pubkey, err)
	}
	return r, nil
}

func ParseIdentity(privkey string) (age.Identity, error) {
	i, err := age.ParseX25519Identity(privkey)
	if err != nil {
		return nil, fmt.Errorf("parse identity: %w", err)
	}
	return i, nil
}

func ParseRecipients(pubkeys []string) ([]age.Recipient, error) {
	var recipients []age.Recipient
	for _, pk := range pubkeys {
		pk = strings.TrimSpace(pk)
		if pk == "" {
			continue
		}
		r, err := ParseRecipient(pk)
		if err != nil {
			return nil, err
		}
		recipients = append(recipients, r)
	}
	return recipients, nil
}

func ParseSSHRecipient(sshPubKey string) (age.Recipient, error) {
	r, err := agessh.ParseRecipient(sshPubKey)
	if err != nil {
		return nil, fmt.Errorf("parse SSH recipient: %w", err)
	}
	return r, nil
}

func ParseSSHRecipients(sshPubKeys []string) ([]age.Recipient, error) {
	var recipients []age.Recipient
	for _, pk := range sshPubKeys {
		pk = strings.TrimSpace(pk)
		if pk == "" {
			continue
		}
		r, err := ParseSSHRecipient(pk)
		if err != nil {
			return nil, err
		}
		recipients = append(recipients, r)
	}
	return recipients, nil
}

func LoadSSHIdentity(homeDir string) (age.Identity, error) {
	paths := []string{
		homeDir + "/.ssh/id_ed25519",
		homeDir + "/.ssh/id_rsa",
	}

	for _, privPath := range paths {
		pemBytes, err := os.ReadFile(privPath)
		if err != nil {
			continue
		}

		identity, err := agessh.ParseIdentity(pemBytes)
		if err != nil {
			pubPath := privPath + ".pub"
			pubBytes, pubErr := os.ReadFile(pubPath)
			if pubErr != nil {
				return nil, fmt.Errorf("SSH key at %s is passphrase-protected but no .pub file found: %w", privPath, pubErr)
			}
			pubKey, _, _, _, parseErr := ssh.ParseAuthorizedKey(pubBytes)
			if parseErr != nil {
				return nil, fmt.Errorf("parse %s: %w", pubPath, parseErr)
			}
			encIdentity, encErr := agessh.NewEncryptedSSHIdentity(pubKey, pemBytes, func() ([]byte, error) {
				return nil, fmt.Errorf("passphrase-protected SSH keys are not supported in non-interactive mode")
			})
			if encErr != nil {
				return nil, fmt.Errorf("load encrypted SSH key: %w", encErr)
			}
			return encIdentity, nil
		}

		return identity, nil
	}

	return nil, fmt.Errorf("no SSH private key found at ~/.ssh/id_ed25519 or ~/.ssh/id_rsa")
}
