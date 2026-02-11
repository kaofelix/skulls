# skulls

Dead simple skills 💀

`skulls` installs skills from git repositories into a target directory.

## Commands

### Interactive search

```bash
skulls --dir <target-dir> [--force]
```

Behavior:
- Empty query shows popular skills.
- Queries with length >= 2 search via `https://skills.sh/api/search`.
- The right pane previews the selected skill's `SKILL.md` (best-effort; GitHub sources only).
- `Enter` installs the selected skill. `Esc` quits.

### Add (direct install / source selector)

```bash
# direct install
skulls add <source> <skill-id> --dir <target-dir>

# shorthand direct install
skulls add owner/repo@skill-id --dir <target-dir>

# interactive selector (when skill-id is omitted)
skulls add <source> --dir <target-dir>
```

`<source>` formats:
- `owner/repo` (GitHub shorthand)
- a git remote URL (`https://...`, `git@...`, `file:///...`)
- a local path to a git repo

Notes:
- When `<skill-id>` is omitted, skulls discovers `skills/**/SKILL.md` in the source and opens an interactive selector.
- In add mode, installs overwrite existing target skill folders.

## Install layout

Installs to:

```
<target-dir>/<skill-id>/
  SKILL.md
  ...
```

Repository layout:
- Skulls discovers skills by scanning `skills/**/SKILL.md` and matching YAML frontmatter `name: <skill-id>`.

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
