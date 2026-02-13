# skulls

Dead simple skills 💀

`skulls` installs skills from git repositories into a target directory.

## Install

### Homebrew

```bash
brew tap kaofelix/tap
brew install skulls
```

## Commands

### Interactive search

```bash
skulls [--dir <target-dir>] [--force]
```

Behavior:
- If `--dir` is omitted, skulls uses the saved install directory from config.
- If no saved directory exists, pass `--dir <target-dir>` for one-off installs or set a default with `skulls config set dir <path>`.
- After installs that use `--dir`, skulls prints a friendly tip on how to persist that directory as your default.
- Empty query shows popular skills.
- Queries with length >= 2 search via `https://skills.sh/api/search`.
- The right pane previews the selected skill's `SKILL.md` (best-effort; GitHub sources only).
- `Enter` installs the selected skill. `Esc` quits.

### Add (direct install / source selector)

```bash
# direct install
skulls add <source> <skill-id> [--dir <target-dir>]

# shorthand direct install
skulls add owner/repo@skill-id [--dir <target-dir>]

# interactive selector (when skill-id is omitted)
skulls add <source> [--dir <target-dir>]
```

`<source>` formats:
- `owner/repo` (GitHub shorthand)
- a git remote URL (`https://...`, `git@...`, `file:///...`)
- a local path to a git repo

Notes:
- When `<skill-id>` is omitted, skulls discovers `skills/**/SKILL.md` in the source and opens an interactive selector.
- In add mode, installs overwrite existing target skill folders.
- `--dir` always overrides the saved config value.

### Config

```bash
skulls config set dir <path>
skulls config get
```

## Install layout

Installs to:

```
<target-dir>/<skill-id>/
  SKILL.md
  ...
```

Repository layout:
- Skulls validates `SKILL.md` frontmatter with required string fields: `name` and `description`.
- Discovery follows Vercel-style priority locations (`skills/`, `skills/.curated/`, `.agent/skills/`, `.claude/skills/`, etc.) and falls back to bounded recursive search.
- A root `SKILL.md` is treated as a direct skill and is preferred by default.

## Development

### Build

```bash
go build ./cmd/skulls
```

### Git hooks (format + lint + test)

This repo uses [prek](https://prek.j178.dev/) (pre-commit compatible).

Install tools:

```bash
# option 1
brew install golangci-lint

# option 2
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

One-time setup per clone:

```bash
./scripts/install-hooks.sh
```

What it does:
- Installs **pre-commit** and **pre-push** git hooks via `prek`.
- On commit: runs `gofmt` (auto-fix), `golangci-lint run`, and `go test ./...`.
- On push: runs `go test ./...`.

Notes:
- The `gofmt` hook will auto-format files. If it modifies files, your commit may be stopped and you'll need to re-stage and commit again.

### Release process (Git tag + Homebrew tap)

1. Create and push a release tag in this repo:

```bash
git tag -a v0.1.1 -m "v0.1.1"
git push origin v0.1.1
```

2. Create the GitHub release:

```bash
gh release create v0.1.1 --title "v0.1.1" --generate-notes
```

3. Update Homebrew formula in the tap repo using the helper script:

```bash
./scripts/update-homebrew-formula.sh v0.1.1 ../homebrew-tap
```

This downloads the release tarball, computes SHA256, and writes:

- `../homebrew-tap/Formula/skulls.rb`

4. Commit and push the tap changes:

```bash
cd ../homebrew-tap
git add Formula/skulls.rb
git commit -m "skulls v0.1.1"
git push
```

5. Verify installation:

```bash
brew update
brew tap kaofelix/tap
brew reinstall skulls
skulls --help
```
