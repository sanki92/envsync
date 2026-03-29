# envsync

Sync encrypted `.env` files through git. No server, no vendor, no shared secrets.

Each value is encrypted with [age](https://age-encryption.org/) using the team's SSH public keys from GitHub. The vault lives in git. Decryption happens automatically on `git pull` via a post-merge hook.

## How it works

```
.env.local  ->  encrypt per-value  ->  .env.vault  ->  git push
                                                            |
                                                        git pull
                                                            |
                                              .env.local auto-updated
```

The encrypted file (`.env.vault`) is safe to commit. The team manifest (`.envteam`) stores SSH public keys and fingerprints. Your SSH private key stays on your machine and never leaves it.

## Install

```bash
go install github.com/sanki92/envsync@latest
```

Build from source:

```bash
git clone https://github.com/sanki92/envsync.git
cd envsync && go build -o envsync .
```

## Setup

**Repo owner (first time):**

```bash
envsync init
git add .env.vault .envteam .gitignore
git commit -m "chore: init envsync"
git push
```

**Adding a teammate:**

```bash
envsync add alice
git add .env.vault .envteam
git commit -m "chore: add alice"
git push
```

That's it. No key exchange needed. envsync fetches their SSH public key directly from GitHub.

**Teammate joins:**

```bash
envsync join        # verifies SSH key setup
git pull            # post-merge hook runs envsync unlock automatically
```

Or manually: `envsync unlock`

## Daily use

```bash
# after editing .env.local
envsync lock
git add .env.vault
git commit -m "chore: update env"
git push

# teammates run git pull, their .env.local updates automatically
```

## Commands

| Command | Description |
|---------|-------------|
| `envsync init` | Set up envsync in a repo |
| `envsync lock` | Encrypt `.env.local` to `.env.vault` |
| `envsync unlock` | Decrypt `.env.vault` to `.env.local` |
| `envsync join` | Verify local SSH setup and install git hook |
| `envsync add <user>` | Add a team member by GitHub username |
| `envsync remove <user>` | Remove a team member and re-encrypt |
| `envsync status` | Show vault state and team |
| `envsync doctor` | Check local setup health |
| `envsync diff` | Show what changed in the vault between commits |

## Prerequisites

Every team member needs an SSH key on their machine and on their GitHub profile. If you use HTTPS/GCM for git, that's fine. The SSH key is only used for encryption, not for git auth.

```bash
# If you don't have an SSH key yet:
ssh-keygen -t ed25519 -C "your-email@example.com"
# Add to GitHub: github.com/settings/keys
```

## Files

| File | Committed | What it holds |
|------|-----------|---------------|
| `.env.vault` | yes | Encrypted values |
| `.envteam` | yes | SSH public keys and fingerprints |
| `.env.local` | no | Plaintext secrets |

## Security notes

- Per-value encryption (not whole-file), so a leaked value doesn't expose others
- Multi-recipient: everyone decrypts independently with their own SSH private key
- SSH key fingerprints are pinned in `.envteam` so key changes are caught
- `NEXT_PUBLIC_*`, `REACT_APP_*`, and `VITE_*` keys are left unencrypted by convention
- Nothing is sent to any server at any point

## License

MIT
