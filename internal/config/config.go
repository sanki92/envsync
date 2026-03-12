package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func Dir() (string, error) {
	home, err := homeDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(home, ".envsync")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	return dir, nil
}

func PrivateKeyPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "private.key"), nil
}

func PublicKeyPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "public.key"), nil
}

func SaveKeypair(privateKey, publicKey string) error {
	privPath, err := PrivateKeyPath()
	if err != nil {
		return err
	}
	pubPath, err := PublicKeyPath()
	if err != nil {
		return err
	}

	if err := os.WriteFile(privPath, []byte(privateKey+"\n"), 0600); err != nil {
		return fmt.Errorf("write private key: %w", err)
	}
	if err := os.WriteFile(pubPath, []byte(publicKey+"\n"), 0644); err != nil {
		return fmt.Errorf("write public key: %w", err)
	}
	return nil
}

func LoadPrivateKey() (string, error) {
	path, err := PrivateKeyPath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("no age identity found at %s — run 'envsync init' or 'envsync join' first", path)
	}

	return string(data), nil
}

func LoadPublicKey() (string, error) {
	path, err := PublicKeyPath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("no age public key found at %s", path)
	}

	return string(data), nil
}

func HasKeypair() bool {
	privPath, err := PrivateKeyPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(privPath)
	return err == nil
}

func homeDir() (string, error) {
	if runtime.GOOS == "windows" {
		home := os.Getenv("USERPROFILE")
		if home != "" {
			return home, nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determine home directory: %w", err)
	}
	return home, nil
}
