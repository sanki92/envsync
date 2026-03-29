package team_test

import (
	"path/filepath"
	"testing"

	"github.com/sanki92/envsync/internal/team"
)

const testSSHKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl test@example.com"
const testSSHKey2 = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBVuKQosNjTpHgSOsDqMYGTR3Sjl3bEQWUNZm6JjqKAR bob@example.com"

func TestNewTeamFile(t *testing.T) {
	tf := team.NewTeamFile("alice", "SHA256:abc123", testSSHKey, "alice")

	if tf.Version != 1 {
		t.Fatalf("expected version 1, got %d", tf.Version)
	}

	m, ok := tf.GetMember("alice")
	if !ok {
		t.Fatal("alice should exist")
	}
	if m.GitHub != "alice" {
		t.Fatalf("expected github alice, got %s", m.GitHub)
	}
	if m.SSHPublicKey != testSSHKey {
		t.Fatalf("unexpected SSH key: %s", m.SSHPublicKey)
	}
}

func TestAddRemoveMember(t *testing.T) {
	tf := team.NewTeamFile("alice", "SHA256:aaa", testSSHKey, "alice")

	if err := tf.AddMember("bob", "SHA256:bbb", testSSHKey2, "alice"); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	if len(tf.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(tf.Members))
	}

	if err := tf.AddMember("bob", "SHA256:bbb", testSSHKey2, "alice"); err == nil {
		t.Fatal("expected error on duplicate add")
	}

	if err := tf.RemoveMember("bob"); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}

	if len(tf.Members) != 1 {
		t.Fatalf("expected 1 member after remove, got %d", len(tf.Members))
	}

	if err := tf.RemoveMember("charlie"); err == nil {
		t.Fatal("expected error removing non-existent member")
	}
}

func TestGetPublicKeys(t *testing.T) {
	tf := team.NewTeamFile("alice", "SHA256:aaa", testSSHKey, "alice")
	tf.AddMember("bob", "SHA256:bbb", testSSHKey2, "alice")

	keys := tf.GetPublicKeys()
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
}

func TestGetSSHPublicKeys(t *testing.T) {
	tf := team.NewTeamFile("alice", "SHA256:aaa", testSSHKey, "alice")
	tf.AddMember("bob", "SHA256:bbb", testSSHKey2, "alice")

	keys := tf.GetSSHPublicKeys()
	if len(keys) != 2 {
		t.Fatalf("expected 2 SSH keys, got %d", len(keys))
	}
}

func TestUsesSSHKeys(t *testing.T) {
	tf := team.NewTeamFile("alice", "SHA256:aaa", testSSHKey, "alice")
	if !tf.UsesSSHKeys() {
		t.Fatal("expected UsesSSHKeys to return true")
	}
}

func TestReadWriteTeamFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".envteam")

	original := team.NewTeamFile("alice", "SHA256:abc", testSSHKey, "alice")
	original.AddMember("bob", "SHA256:def", testSSHKey2, "alice")

	if err := team.WriteTeamFile(path, original); err != nil {
		t.Fatalf("WriteTeamFile: %v", err)
	}

	parsed, err := team.ReadTeamFile(path)
	if err != nil {
		t.Fatalf("ReadTeamFile: %v", err)
	}

	if parsed.Version != 1 {
		t.Fatalf("version: got %d, want 1", parsed.Version)
	}
	if len(parsed.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(parsed.Members))
	}

	alice, ok := parsed.GetMember("alice")
	if !ok {
		t.Fatal("alice not found after read")
	}
	if alice.SSHFingerprint != "SHA256:abc" {
		t.Fatalf("alice fingerprint: %s", alice.SSHFingerprint)
	}
	if alice.SSHPublicKey != testSSHKey {
		t.Fatalf("alice SSH key not persisted: %s", alice.SSHPublicKey)
	}
}

func TestMemberNames(t *testing.T) {
	tf := team.NewTeamFile("alice", "", "", "alice")
	tf.AddMember("bob", "", "", "alice")

	names := tf.MemberNames()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
}

func TestEnvironmentAccess(t *testing.T) {
	tf := team.NewTeamFile("alice", "SHA256:aaa", testSSHKey, "alice")
	tf.AddMember("bob", "SHA256:bbb", testSSHKey2, "alice")

	tf.AddEnvToMember("bob", "staging")

	defaultMembers := tf.MemberNamesForEnv("")
	if len(defaultMembers) != 1 {
		t.Fatalf("expected 1 default member (alice), got %d: %v", len(defaultMembers), defaultMembers)
	}

	stagingMembers := tf.MemberNamesForEnv("staging")
	if len(stagingMembers) != 1 {
		t.Fatalf("expected 1 staging member (bob), got %d: %v", len(stagingMembers), stagingMembers)
	}
	if stagingMembers[0] != "bob" {
		t.Fatalf("expected bob in staging, got %s", stagingMembers[0])
	}
}

func TestAddRemoveEnv(t *testing.T) {
	tf := team.NewTeamFile("alice", "SHA256:aaa", testSSHKey, "alice")
	tf.AddMember("bob", "SHA256:bbb", testSSHKey2, "alice")

	tf.AddEnvToMember("bob", "staging")
	tf.AddEnvToMember("bob", "production")

	m, _ := tf.GetMember("bob")
	if len(m.Environments) != 2 {
		t.Fatalf("expected 2 envs, got %d", len(m.Environments))
	}

	tf.AddEnvToMember("bob", "staging")
	m, _ = tf.GetMember("bob")
	if len(m.Environments) != 2 {
		t.Fatalf("duplicate add should be no-op, got %d envs", len(m.Environments))
	}

	tf.RemoveEnvFromMember("bob", "staging")
	m, _ = tf.GetMember("bob")
	if len(m.Environments) != 1 {
		t.Fatalf("expected 1 env after remove, got %d", len(m.Environments))
	}
}
