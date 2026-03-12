# @sanki92/envsync

Use envsync without installing Go manually. The npm package installs a small wrapper that downloads the right envsync binary from GitHub Releases on first run and then executes it locally.

The binary is cached in `~/.envsync/bin`.

## Install

Run once without installing globally:

```bash
npx @sanki92/envsync init
```

Install globally:

```bash
npm install -g @sanki92/envsync
envsync --help
```

The package name is `@sanki92/envsync`, but the installed command is `envsync`.

## What it does

envsync keeps encrypted `.env` files in git for a team.

- `envsync lock` encrypts `.env.local` into `.env.vault`
- `envsync unlock` decrypts `.env.vault` into `.env.local`
- `envsync init` sets up a repo for envsync
- `envsync join` prints the age public key a teammate shares with the repo admin
- `envsync add` and `envsync remove` manage team access

Each value is encrypted separately with age. Team access is based on GitHub SSH public keys.

## Quick start

Initialize a repo:

```bash
envsync init
git add .env.vault .envteam .gitignore
git commit -m "chore: init envsync"
git push
```

Join an existing team:

```bash
envsync join
```

After the admin adds your printed age public key to the team, pull and decrypt:

```bash
git pull
envsync unlock
```

## Supported platforms

- Linux amd64
- Linux arm64
- macOS amd64
- macOS arm64
- Windows amd64
- Windows arm64

## Requirements

- A GitHub repo for your project
- Git installed locally
- A GitHub SSH key on each developer machine

## Notes

- The first run downloads the matching binary from the GitHub release for this package version
- Nothing is sent to any envsync server because there is no envsync server
- If you prefer, you can install from source with `go install github.com/sanki92/envsync@latest`

## Links

- Source: https://github.com/sanki92/envsync
- Releases: https://github.com/sanki92/envsync/releases