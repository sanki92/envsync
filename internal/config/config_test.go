package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sanki92/envsync/internal/config"
	"github.com/sanki92/envsync/internal/crypto"
)

func TestSaveLoadKeypair(t *testing.T) {
	priv, pub, err := crypto.GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair: %v", err)
	}

	if err := config.SaveKeypair(priv, pub); err != nil {
		t.Fatalf("SaveKeypair: %v", err)
	}

	loadedPriv, err := config.LoadPrivateKey()
	if err != nil {
		t.Fatalf("LoadPrivateKey: %v", err)
	}
	if strings.TrimSpace(loadedPriv) != priv {
		t.Fatalf("private key mismatch")
	}

	loadedPub, err := config.LoadPublicKey()
	if err != nil {
		t.Fatalf("LoadPublicKey: %v", err)
	}
	if strings.TrimSpace(loadedPub) != pub {
		t.Fatalf("public key mismatch")
	}
}

func TestHasKeypair(t *testing.T) {
	_ = config.HasKeypair()
}

func TestDir(t *testing.T) {
	dir, err := config.Dir()
	if err != nil {
		t.Fatalf("Dir: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat config dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("config dir should be a directory")
	}

	if !strings.HasSuffix(filepath.ToSlash(dir), ".envsync") {
		t.Fatalf("unexpected dir: %s", dir)
	}
}
