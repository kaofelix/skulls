# Skulls plan

This project is built as **vertical slices**. Each slice should be runnable end-to-end and feel like a usable increment.

## Product goals

- **Dead simple**: pick a target directory, install skills into it.
- **Search-first UX**: running `skulls` with no arguments opens an interactive full-screen search UI.
- **Review before installing**: preview `SKILL.md` rendered as highlighted Markdown.

## Status

### Completed slices

1. **Installer** ✅  
   `skulls add <source> <skill-id> [--dir <target-dir>] [--force]`
2. **Full-screen TUI search mode** ✅  
   `skulls [--dir <target-dir>] [--force]`
3. **Popular-by-default** ✅
4. **Preview pane with highlighted Markdown** ✅
5. **Add mode source selector** ✅  
   `skulls add <source> [skill-id] [--dir <target-dir>]`
6. **Remember target directory** ✅  
   - First run prompts for dir and persists it
   - `skulls config set dir <path>`
   - `skulls config get`
   - `--dir` always overrides

## Current scope (post-slices hardening)

- **Collision handling UX**: overwrite / rename / cancel.

- **Better repo layout detection (next):**
  - Plugin-manifest discovery parity:
    - `.claude-plugin/marketplace.json`
    - `.claude-plugin/plugin.json`
  - Optional full-depth discovery flag for CLI flows (root + nested skills).
  - Better ambiguity diagnostics when multiple paths map to the same skill name.
  - GitHub preview fallback hardening when Trees API is truncated.
  - Explicit subpath-first discovery mode for unusual monorepos.
  - Additional safety checks around symlinks/path traversal during discovery.
  - Fixture-based cross-layout test corpus.

- **Support additional source formats safely**.
