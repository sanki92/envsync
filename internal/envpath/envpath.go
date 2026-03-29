package envpath

import "path/filepath"

func VaultPath(repoRoot, env string) string {
	if env == "" || env == "development" {
		return filepath.Join(repoRoot, ".env.vault")
	}
	return filepath.Join(repoRoot, ".env."+env+".vault")
}

func LocalPath(repoRoot, env string) string {
	if env == "" || env == "development" {
		return filepath.Join(repoRoot, ".env.local")
	}
	return filepath.Join(repoRoot, ".env."+env+".local")
}

func VaultFilename(env string) string {
	if env == "" || env == "development" {
		return ".env.vault"
	}
	return ".env." + env + ".vault"
}

func LocalFilename(env string) string {
	if env == "" || env == "development" {
		return ".env.local"
	}
	return ".env." + env + ".local"
}
