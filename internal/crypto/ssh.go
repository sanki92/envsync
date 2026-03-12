package crypto

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

type SSHKeyInfo struct {
	Type        string
	Fingerprint string
	Comment     string
}

func FingerprintSSHPublicKey(authorizedKey string) (*SSHKeyInfo, error) {
	pubKey, comment, _, _, err := ssh.ParseAuthorizedKey([]byte(authorizedKey))
	if err != nil {
		return nil, fmt.Errorf("parse SSH public key: %w", err)
	}

	hash := sha256.Sum256(pubKey.Marshal())
	fingerprint := "SHA256:" + base64.RawStdEncoding.EncodeToString(hash[:])

	return &SSHKeyInfo{
		Type:        pubKey.Type(),
		Fingerprint: fingerprint,
		Comment:     comment,
	}, nil
}

func ReadLocalSSHPublicKey(homeDir string) (string, string, error) {
	paths := []string{
		homeDir + "/.ssh/id_ed25519.pub",
		homeDir + "/.ssh/id_rsa.pub",
	}

	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		content := strings.TrimSpace(string(data))
		if content != "" {
			return content, p, nil
		}
	}

	return "", "", fmt.Errorf("no SSH public key found at ~/.ssh/id_ed25519.pub or ~/.ssh/id_rsa.pub")
}

func HasSSHPrivateKey(homeDir string) (string, bool) {
	paths := []string{
		homeDir + "/.ssh/id_ed25519",
		homeDir + "/.ssh/id_rsa",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	return "", false
}

func HasSSHPassphrase(keyPath string) (bool, error) {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return false, fmt.Errorf("read private key: %w", err)
	}

	return strings.Contains(string(data), "ENCRYPTED"), nil
}
