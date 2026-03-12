package crypto

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"filippo.io/age"
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
