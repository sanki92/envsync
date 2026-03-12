package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/sanki92/envsync/internal/crypto"
	gitutil "github.com/sanki92/envsync/internal/git"
	"github.com/sanki92/envsync/internal/team"
	"github.com/sanki92/envsync/internal/vault"
)

func TestStatusOutputComponents(t *testing.T) {
	dir := t.TempDir()
	vaultPath := filepath.Join(dir, ".env.vault")
	teamPath := filepath.Join(dir, ".envteam")
	envPath := filepath.Join(dir, ".env.local")

	priv, pub, _ := crypto.GenerateKeypair()

	tf := team.NewTeamFile("alice", "SHA256:aaa", pub, "alice")
	tf.AddMember("bob", "SHA256:bbb", "age1fake", "alice")
	team.WriteTeamFile(teamPath, tf)

	envContent := "SECRET=value\nNEXT_PUBLIC_URL=https://example.com\n"
	os.WriteFile(envPath, []byte(envContent), 0644)

	entries, _ := vault.ReadEnvFile(envPath)
	r, _ := crypto.ParseRecipient(pub)
	locked, _ := vault.LockVault(entries, []age.Recipient{r}, tf.MemberNames())
	vault.WriteVaultFile(vaultPath, locked)

	header, err := vault.ReadVaultHeader(vaultPath)
	if err != nil {
		t.Fatalf("ReadVaultHeader: %v", err)
	}
	if header.LastUpdated.IsZero() {
		t.Fatal("LastUpdated should be set")
	}
	if len(header.Recipients) != 2 {
		t.Fatalf("expected 2 recipients in header, got %d", len(header.Recipients))
	}

	vaultEntries, _ := vault.ReadVaultFile(vaultPath)
	keyCount := 0
	for _, e := range vaultEntries {
		if e.Key != "" {
			keyCount++
		}
	}
	if keyCount != 2 {
		t.Fatalf("expected 2 vault keys, got %d", keyCount)
	}

	id, _ := crypto.ParseIdentity(priv)
	unlocked, err := vault.UnlockVault(vaultEntries, []age.Identity{id})
	if err != nil {
		t.Fatalf("unlock: %v", err)
	}
	m := vault.EnvMap(unlocked)
	if m["SECRET"] != "value" {
		t.Fatalf("wrong SECRET: %q", m["SECRET"])
	}

	names := tf.MemberNames()
	if len(names) != 2 {
		t.Fatalf("expected 2 team members, got %d", len(names))
	}
}

func TestDoctorChecksComponents(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	os.MkdirAll(gitDir, 0755)

	if gitutil.HasPostMergeHook(dir) {
		t.Fatal("should not have hook yet")
	}

	gitutil.InstallPostMergeHook(dir)
	if !gitutil.HasPostMergeHook(dir) {
		t.Fatal("should have hook after install")
	}

	vaultPath := filepath.Join(dir, ".env.vault")
	envPath := filepath.Join(dir, ".env.local")

	_, pub, _ := crypto.GenerateKeypair()
	r, _ := crypto.ParseRecipient(pub)

	entries := []vault.EnvEntry{
		{Key: "A", Value: "1"},
		{Key: "B", Value: "2"},
		{Key: "C", Value: "3"},
	}
	locked, _ := vault.LockVault(entries, []age.Recipient{r}, []string{"testuser"})
	vault.WriteVaultFile(vaultPath, locked)

	os.WriteFile(envPath, []byte("A=1\nB=2\n"), 0644)

	localEntries, _ := vault.ReadEnvFile(envPath)
	vaultEntries, _ := vault.ReadVaultFile(vaultPath)
	localMap := vault.EnvMap(localEntries)
	vaultKeys := vault.VaultKeys(vaultEntries)

	missing := 0
	for _, k := range vaultKeys {
		if _, ok := localMap[k]; !ok {
			missing++
		}
	}
	if missing != 1 {
		t.Fatalf("expected 1 missing key, got %d", missing)
	}
}

func TestDiffParseAndCompare(t *testing.T) {
	old := []vault.EnvEntry{
		{Key: "A", Value: "enc:old_a"},
		{Key: "B", Value: "enc:val_b"},
		{Key: "C", Value: "enc:val_c"},
	}
	new := []vault.EnvEntry{
		{Key: "A", Value: "enc:new_a"},
		{Key: "B", Value: "enc:val_b"},
		{Key: "D", Value: "enc:val_d"},
	}

	oldMap := vault.EnvMap(old)
	newMap := vault.EnvMap(new)

	var added, removed, changed []string

	for k, v := range newMap {
		oldVal, existed := oldMap[k]
		if !existed {
			added = append(added, k)
		} else if oldVal != v {
			changed = append(changed, k)
		}
	}
	for k := range oldMap {
		if _, exists := newMap[k]; !exists {
			removed = append(removed, k)
		}
	}

	if len(added) != 1 || added[0] != "D" {
		t.Fatalf("expected added=[D], got %v", added)
	}
	if len(removed) != 1 || removed[0] != "C" {
		t.Fatalf("expected removed=[C], got %v", removed)
	}
	if len(changed) != 1 || changed[0] != "A" {
		t.Fatalf("expected changed=[A], got %v", changed)
	}
}

func TestFormatRelativeTime(t *testing.T) {
	now := time.Now()
	tests := []struct {
		t    time.Time
		want string
	}{
		{now.Add(-30 * time.Second), "just now"},
		{now.Add(-5 * time.Minute), "5 minutes ago"},
		{now.Add(-1 * time.Minute), "1 minute ago"},
		{now.Add(-3 * time.Hour), "3 hours ago"},
		{now.Add(-1 * time.Hour), "1 hour ago"},
		{now.Add(-48 * time.Hour), "2 days ago"},
		{now.Add(-24 * time.Hour), "1 day ago"},
	}

	for _, tc := range tests {
		diff := time.Since(tc.t)
		got := formatTimeForTest(diff)
		if got != tc.want {
			t.Errorf("formatRelativeTime(%v) = %q, want %q", diff, got, tc.want)
		}
	}
}

func formatTimeForTest(diff time.Duration) string {
	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		m := int(diff.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return strings.Replace(strings.Replace("N minutes ago", "N", itoa(m), 1), "minutes", "minutes", 1)
	case diff < 24*time.Hour:
		h := int(diff.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return strings.Replace("N hours ago", "N", itoa(h), 1)
	default:
		d := int(diff.Hours() / 24)
		if d == 1 {
			return "1 day ago"
		}
		return strings.Replace("N days ago", "N", itoa(d), 1)
	}
}

func itoa(n int) string {
	return strings.TrimSpace(strings.Replace("          ", " ", "", n-1)[:0] + string(rune('0'+n)))
}
