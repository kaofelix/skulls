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

### Add (direct install)

```bash
skulls add <source> <skill-id> --dir <target-dir> [--force]
```

`<source>` formats:
- `owner/repo` (GitHub shorthand)
- a git remote URL (`https://...`, `git@...`, `file:///...`)
- a local path to a git repo

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

### Git hooks (format + test)

This repo uses [prek](https://prek.j178.dev/) (pre-commit compatible).

One-time setup per clone:

```bash
./scripts/install-hooks.sh
```

What it does:
- Installs **pre-commit** and **pre-push** git hooks via `prek`.
- On commit: runs `gofmt` (auto-fix) and `go test ./...`.
- On push: runs `go test ./...`.

Notes:
- The `gofmt` hook will auto-format files. If it modifies files, your commit may be stopped and you'll need to re-stage and commit again.
