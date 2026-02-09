# skulls

Dead simple skills 💀

## Status

This repo is being built incrementally as vertical slices.

See [PLAN.md](./PLAN.md) for the tracked plan and upcoming slices.

## Git hooks (format + test)

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
- The `gofmt` hook will auto-format files. If it modifies files, your commit may be stopped and you’ll need to re-stage and commit again.

## Slice 1 (current): install a skill into any target directory

```bash
# build
go build ./cmd/skulls

# install a skill
./skulls add obra/superpowers using-git-worktrees --dir ~/.pi/agent/skills

# overwrite
./skulls add obra/superpowers using-git-worktrees --dir ~/.pi/agent/skills --force
```

Notes:
- For now, Skulls expects skills to live at `skills/<skill-id>/SKILL.md` in the repo.
- Running `skulls` with no args will eventually open the interactive search UI.
