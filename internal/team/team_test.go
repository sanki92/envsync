package team_test

import (
	"path/filepath"
	"testing"

	"github.com/sanki92/envsync/internal/team"
)

func TestNewTeamFile(t *testing.T) {
	tf := team.NewTeamFile("alice", "SHA256:abc123", "age1xyz...", "alice")

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
	if m.AgePublicKey != "age1xyz..." {
		t.Fatalf("unexpected age key: %s", m.AgePublicKey)
	}
}

func TestAddRemoveMember(t *testing.T) {
	tf := team.NewTeamFile("alice", "SHA256:aaa", "age1aaa", "alice")

	if err := tf.AddMember("bob", "SHA256:bbb", "age1bbb", "alice"); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	if len(tf.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(tf.Members))
	}

	if err := tf.AddMember("bob", "SHA256:bbb", "age1bbb", "alice"); err == nil {
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
	tf := team.NewTeamFile("alice", "SHA256:aaa", "age1aaa", "alice")
	tf.AddMember("bob", "SHA256:bbb", "age1bbb", "alice")

	keys := tf.GetPublicKeys()
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
}

func TestReadWriteTeamFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".envteam")

	original := team.NewTeamFile("alice", "SHA256:abc", "age1abc", "alice")
	original.AddMember("bob", "SHA256:def", "age1def", "alice")

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
}

func TestMemberNames(t *testing.T) {
	tf := team.NewTeamFile("alice", "", "", "alice")
	tf.AddMember("bob", "", "", "alice")

	names := tf.MemberNames()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
}
