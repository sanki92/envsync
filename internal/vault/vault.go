package vault

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/sanki92/envsync/internal/crypto"
)

const EncPrefix = "enc:"

type VaultHeader struct {
	Recipients  []string
	LastUpdated time.Time
}

func LockVaultSSH(entries []EnvEntry, sshPubKeys []string, memberNames []string) ([]EnvEntry, error) {
	recipients, err := crypto.ParseSSHRecipients(sshPubKeys)
	if err != nil {
		return nil, fmt.Errorf("parse SSH recipients: %w", err)
	}
	return LockVault(entries, recipients, memberNames)
}

func UnlockVaultSSH(vaultEntries []EnvEntry, homeDir string) ([]EnvEntry, error) {
	identity, err := crypto.LoadSSHIdentity(homeDir)
	if err != nil {
		return nil, fmt.Errorf("load SSH identity: %w", err)
	}
	return UnlockVault(vaultEntries, []age.Identity{identity})
}

func LockVault(entries []EnvEntry, recipients []age.Recipient, memberNames []string) ([]EnvEntry, error) {
	header := []EnvEntry{
		{Comment: "# envsync vault - safe to commit"},
		{Comment: fmt.Sprintf("# Recipients: %s", strings.Join(memberNames, ", "))},
		{Comment: fmt.Sprintf("# Last updated: %s", time.Now().UTC().Format(time.RFC3339))},
		{Comment: ""},
	}

	var vaultEntries []EnvEntry
	vaultEntries = append(vaultEntries, header...)

	for _, e := range entries {
		if e.Key == "" {
			continue
		}

		if IsPublicKey(e.Key) {
			vaultEntries = append(vaultEntries, EnvEntry{Key: e.Key, Value: e.Value})
			continue
		}

		ciphertext, err := crypto.Encrypt(e.Value, recipients)
		if err != nil {
			return nil, fmt.Errorf("encrypt %s: %w", e.Key, err)
		}
		vaultEntries = append(vaultEntries, EnvEntry{Key: e.Key, Value: EncPrefix + ciphertext})
	}

	return vaultEntries, nil
}

func UnlockVault(vaultEntries []EnvEntry, identities []age.Identity) ([]EnvEntry, error) {
	var entries []EnvEntry

	for _, e := range vaultEntries {
		if e.Key == "" {
			continue
		}

		if !strings.HasPrefix(e.Value, EncPrefix) {
			entries = append(entries, EnvEntry{Key: e.Key, Value: e.Value})
			continue
		}

		ciphertext := strings.TrimPrefix(e.Value, EncPrefix)
		plaintext, err := crypto.Decrypt(ciphertext, identities)
		if err != nil {
			return nil, fmt.Errorf("decrypt %s: %w", e.Key, err)
		}
		entries = append(entries, EnvEntry{Key: e.Key, Value: plaintext})
	}

	return entries, nil
}

func ReadVaultFile(path string) ([]EnvEntry, error) {
	return ReadEnvFile(path)
}

func WriteVaultFile(path string, entries []EnvEntry) error {
	return WriteEnvFile(path, entries)
}

func VaultKeys(entries []EnvEntry) []string {
	var keys []string
	for _, e := range entries {
		if e.Key != "" {
			keys = append(keys, e.Key)
		}
	}
	return keys
}

func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, EncPrefix)
}

func ReadVaultHeader(path string) (*VaultHeader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	header := &VaultHeader{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "#") {
			break
		}

		if strings.HasPrefix(line, "# Recipients:") {
			names := strings.TrimPrefix(line, "# Recipients:")
			for _, n := range strings.Split(names, ",") {
				n = strings.TrimSpace(n)
				if n != "" {
					header.Recipients = append(header.Recipients, n)
				}
			}
		}

		if strings.HasPrefix(line, "# Last updated:") {
			ts := strings.TrimSpace(strings.TrimPrefix(line, "# Last updated:"))
			if t, err := time.Parse(time.RFC3339, ts); err == nil {
				header.LastUpdated = t
			}
		}
	}

	return header, scanner.Err()
}
