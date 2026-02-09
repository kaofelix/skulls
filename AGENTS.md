# Working agreement (Skulls)

This repository is built in **vertical slices** (see [PLAN.md](./PLAN.md)). For each slice, follow this approach.

## Slice workflow

1. **Analyze & plan**
   - Review the slice requirements and existing code.
   - Propose an implementation plan focused on an end-to-end deliverable.
   - Call out risks, unknowns, and any UX questions.
   - **Ask the user for feedback/approval** before implementing.

2. **Implement with TDD**
   - Use test-driven development for new functionality.
   - Start from a failing test, implement the smallest change to pass, iterate.
   - Keep going until the slice is **feature complete**.

3. **End-to-end verification**
   - When possible, do real CLI runs that exercise the full flow.
   - Prefer black-box checks (execute the binary, verify filesystem output, etc.).

4. **Report back & ask for feedback**
   - Summarize what changed (paths, commands, behavior).
   - Describe how to run/verify it.
   - Ask for user feedback and adjust based on it.

5. **Commits**
   - Do **not** commit by default.
   - **Only commit when the user explicitly asks**.
