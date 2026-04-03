---
schema_version: 1
id: ci/tag-driven-github-releases
title: Tag-Driven GitHub Releases
status: active
aliases:
    - agent-os/standards/ci/tag-driven-github-releases
---

# Tag-Driven GitHub Releases

- Official releases ship only from `.github/workflows/release.yml`
- Trigger only from signed `v*` tags that match `nix eval --raw .#lib.release.tag`
- Never publish official releases manually in the GitHub UI
- Require protected `release` environment approval before publish
- Build unsigned assets first, then sign and attest the exact published files
- Publish the same verified asset set for all supported targets
- Reject sentinel dev tags like `v0.0.0-dev`

```yaml
on:
  push:
    tags: ["v*"]
```
