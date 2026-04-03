---
schema_version: 1
id: ci/worktree-cleanliness
title: CI Worktree Cleanliness
status: active
suggested_context_bundles:
    - ci-tooling
    - runner-boundary
aliases:
    - agent-os/standards/ci/worktree-cleanliness
---

# CI Worktree Cleanliness

- CI must not mutate the repo after the check entrypoint runs (typically: `just ci`)
- After the CI command, fail on:
  - tracked diffs: `git diff --exit-code`
  - untracked files: `git ls-files --others --exclude-standard`
- If you also enforce a lockfile invariant (example: `flake.lock`), run that check first (pre/post) so failures are clear

```sh
git diff --exit-code
untracked="$(git ls-files --others --exclude-standard)"
[ -z "$untracked" ] || { echo "$untracked"; exit 1; }
```
