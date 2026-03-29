package envpath_test

import (
	"testing"

	"github.com/sanki92/envsync/internal/envpath"
)

func TestVaultPath(t *testing.T) {
	tests := []struct {
		env  string
		want string
	}{
		{"", ".env.vault"},
		{"development", ".env.vault"},
		{"staging", ".env.staging.vault"},
		{"production", ".env.production.vault"},
	}
	for _, tt := range tests {
		got := envpath.VaultFilename(tt.env)
		if got != tt.want {
			t.Errorf("VaultFilename(%q) = %q, want %q", tt.env, got, tt.want)
		}
	}
}

func TestLocalPath(t *testing.T) {
	tests := []struct {
		env  string
		want string
	}{
		{"", ".env.local"},
		{"development", ".env.local"},
		{"staging", ".env.staging.local"},
		{"production", ".env.production.local"},
	}
	for _, tt := range tests {
		got := envpath.LocalFilename(tt.env)
		if got != tt.want {
			t.Errorf("LocalFilename(%q) = %q, want %q", tt.env, got, tt.want)
		}
	}
}
