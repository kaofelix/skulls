# skulls

Dead simple skills 💀

Install AI agent skills from git repositories into a directory you control.

## Why skulls?

Skulls is a simpler alternative to [Vercel's skills client](https://github.com/vercel-labs/skills). Vercel's client does a lot: it installs skills into multiple agent-specific directories and manages symlinks to keep them in sync.

I wanted something more straightforward:

1. **Point to a directory** — that's where skills go
2. **Install skills** — they're just folders with a `SKILL.md`
3. **Done** — no symlinks, no magic, no surprises

Most coding agents let you configure custom skill directories anyway. If you need symlinks for multiple agents, you can manage that yourself and know exactly what's happening.

## Install

```bash
brew tap kaofelix/tap
brew install skulls
```

## Quick Start

**1. Set your skills directory (once):**

```bash
skulls config set dir ~/.pi/agent/skills
```

**2. Search and install:**

```bash
skulls
```

This opens an interactive search UI. Type to search, use arrow keys to browse, and press Enter to install.

That's it. Your skill is now in `~/.pi/agent/skills/<skill-name>/`.

## Usage

### Interactive Search

```bash
skulls
```

Opens a full-screen TUI where you can:

- **Search** skills from [skills.sh](https://skills.sh) (type at least 2 characters)
- **Browse popular** skills (shown by default when the search is empty)
- **Preview** the selected skill's `SKILL.md` in the right pane
- **Install** with Enter, quit with Esc

Navigation:
- `↑`/`↓` — select skill
- `Enter` — install selected skill
- `Esc` or `Ctrl+C` — quit
- `PgUp`/`PgDown`, `Ctrl+U`/`Ctrl+D` — scroll preview

### Direct Install

When you know exactly what you want:

```bash
# Install a specific skill from a repo
skulls add owner/repo skill-id

# Shorthand (same as above)
skulls add owner/repo@skill-id

# Examples
skulls add anthropics/skills pdf
skulls add obra/superpowers@test-driven-development
```

When you want to browse a repo's skills:

```bash
# Opens interactive selector for skills in the repo
skulls add owner/repo
skulls add https://github.com/owner/repo.git
skulls add ./local/repo
```

### Configuration

```bash
# Set default install directory
skulls config set dir ~/.pi/agent/skills

# View current config
skulls config get
```

The `--dir` flag overrides the saved directory for a single install:

```bash
skulls --dir /tmp/test-skills
skulls add owner/repo skill-id --dir /tmp/test-skills
```

### Overwriting Existing Skills

By default, skulls won't overwrite an existing skill folder. Use `--force` to replace it:

```bash
skulls --force
skulls add owner/repo skill-id --force
```

## How Skills Are Found

Skulls discovers skills in repositories using a priority-based search:

1. **Root `SKILL.md`** — if the repo itself is a skill
2. **Standard directories** — `skills/`, `.claude/skills/`, `.agent/skills/`, and [20+ other common locations](https://github.com/vercel-labs/skills)
3. **Recursive fallback** — searches up to 5 levels deep

A valid skill requires a `SKILL.md` with YAML frontmatter containing `name` and `description`:

```markdown
---
name: my-skill
description: What this skill does
---

# My Skill

Instructions for the AI agent...
```

## Development

```bash
make build    # build ./skulls binary
make test     # run all tests
make lint     # run golangci-lint
make clean    # remove the binary
make install  # install to GOPATH/bin
```

### Git Hooks

This repo uses [prek](https://prek.j178.dev/) for pre-commit hooks:

```bash
# Install linter
brew install golangci-lint

# Set up hooks (once per clone)
./scripts/install-hooks.sh
```

### Release Process

```bash
# 1. Tag and push
git tag -a v0.1.1 -m "v0.1.1"
git push origin v0.1.1

# 2. Create GitHub release
gh release create v0.1.1 --title "v0.1.1" --generate-notes

# 3. Update Homebrew formula
./scripts/update-homebrew-formula.sh v0.1.1 ../homebrew-tap

# 4. Push tap changes
cd ../homebrew-tap
git add Formula/skulls.rb
git commit -m "skulls v0.1.1"
git push
```
