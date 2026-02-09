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

## Slices 1–2: install + interactive search UI

```bash
# build
go build ./cmd/skulls

# interactive search + install (requires --dir for now)
./skulls --dir ~/.pi/agent/skills

# direct install
./skulls add obra/superpowers using-git-worktrees --dir ~/.pi/agent/skills

# overwrite
./skulls add obra/superpowers using-git-worktrees --dir ~/.pi/agent/skills --force
```

Notes:
- For now, Skulls expects skills to live at `skills/<skill-id>/SKILL.md` in the repo.
- In search mode, Skulls shows a small install progress UI and then exits with a final “Installed …” message.
