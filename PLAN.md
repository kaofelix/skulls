# Skulls plan

This project is built as **vertical slices**: each slice should be runnable end-to-end and feel like a usable increment.

## Product goals

- **Dead simple**: pick a target directory, install skills into it.
- **Search-first UX**: running `skulls` with no arguments opens an interactive, full-screen search UI.
- **Review before installing**: preview the skill’s `SKILL.md` rendered as highlighted Markdown.

## Non-goals (for now)

- Agent-to-directory mappings (Skulls does not care which agent you use).
- Complex repository layouts beyond the common `skills/<skill-id>/SKILL.md` convention.

## Slice 1 — Installer (done)

**CLI**

```bash
skulls add <source> <skill-id> --dir <target-dir> [--force]
```

**Behavior**
- Clone repo shallow (`git clone --depth 1`) into a temp dir.
- Expect `skills/<skill-id>/SKILL.md`.
- Copy `skills/<skill-id>/` → `<target-dir>/<skill-id>/` (flat layout).
- `--force` overwrites an existing target directory.

**Notes**
- `--dir` required for now.

## Slice 2 — Full-screen TUI search mode (done)

**Goal**: `skulls` (no args) launches an interactive search UI.

**UX**
- Top: search input.
- Left: results list.
- Bottom: key hints.
- Query length >= 2 triggers API search:
  - `GET https://skills.sh/api/search?q=<query>&limit=10`
- `Enter` installs the currently selected result using Slice 1 installer.
- `Esc` quits.

**Deliverable**
- End-to-end flow: open → search → select → install.

## Slice 3 — Popular-by-default (done)

**Goal**: show popular skills immediately on open (empty query).

**Approach**
- Fetch `https://skills.sh`.
- Scrape the embedded `initialSkills` list from the HTML payload.
- Sort by `installs` descending.
- Display top N when query is empty.
- Switch to `/api/search` results when query length >= 2.

**Tests**
- Unit test parser against a saved HTML fixture.

## Slice 4 — Preview pane with highlighted Markdown (done)

**Goal**: add a right-side preview pane rendering `SKILL.md`.

**UX**
- Layout: search bar (top), list (left), preview (right).
- On selection change: asynchronously load preview with caching.

**Preview fetching (best-effort)**
- GitHub-only for now.
- Raw fetch using the special ref `HEAD` (resolves to the default branch):
  - `https://raw.githubusercontent.com/<owner>/<repo>/HEAD/skills/<skill-id>/SKILL.md`
- If it fails: show “Preview unavailable”; allow install anyway (clone will still work).

**Rendering**
- Render Markdown → ANSI using Glamour.

## Slice 5 — Remember the target directory

**Goal**: don’t require `--dir` every time.

**UX**
- First run without configured dir: prompt for install dir and persist it.
- Commands:
  - `skulls config set dir <path>`
  - `skulls config get`
- `--dir` always overrides.

## Hardening / polish (post-slices)

- Collision handling: overwrite / rename / cancel.
- Better repo layout detection when `skills/<skill-id>` doesn’t exist (search for `SKILL.md` and match frontmatter name).
- Support more source formats safely.
- Telemetry: likely no (unless explicitly desired).
