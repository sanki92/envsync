# envsync

Sync encrypted `.env` files through git. No server, no vendor, no shared secrets.

Each value is encrypted with [age](https://age-encryption.org/) using the team's GitHub SSH public keys. The vault lives in git. Decryption happens automatically on `git pull` via a post-merge hook.

## How it works

```
.env.local  ->  encrypt per-value  ->  .env.vault  ->  git push
                                                            |
                                                        git pull
                                                            |
                                              .env.local auto-updated
```

The encrypted file (`.env.vault`) is safe to commit. The team manifest (`.envteam`) stores public keys and SSH fingerprints. Your private key stays on your machine and never leaves it.

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

Teammate runs this and shares the age public key it prints:

```bash
envsync join
```

Admin then adds them:

```bash
envsync add alice --key age1...
git add .env.vault .envteam
git commit -m "chore: add alice"
git push
```

**Teammate pulls and decrypts:**

```bash
git pull
# post-merge hook runs envsync unlock automatically
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
| `envsync join` | Generate local keypair, print public key to share with admin |
| `envsync add <user> --key <age-key>` | Add a team member and re-encrypt |
| `envsync remove <user>` | Remove a team member and re-encrypt |
| `envsync status` | Show vault state and team |
| `envsync doctor` | Check local setup health |
| `envsync diff` | Show what changed in the vault between commits |

## Files

| File | Committed | What it holds |
|------|-----------|---------------|
| `.env.vault` | yes | Encrypted values |
| `.envteam` | yes | Usernames and public keys |
| `.env.local` | no | Plaintext secrets |
| `~/.envsync/` | no | Your age keypair |

## Security notes

- Per-value encryption (not whole-file), so a leaked value doesn't expose others
- Multi-recipient: everyone decrypts independently with their own private key
- SSH key fingerprints are pinned in `.envteam` so key changes are caught
- `NEXT_PUBLIC_*`, `REACT_APP_*`, and `VITE_*` keys are left unencrypted by convention
- Nothing is sent to any server at any point

## License

MIT
