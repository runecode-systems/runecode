---
schema_version: 1
id: security/release-install-verification
title: Release Install Verification
status: active
aliases:
    - agent-os/standards/security/release-install-verification
---

# Release Install Verification

- Official install docs show verification before install
- Prefer copyable manual commands over `curl | bash`
- Resolve `latest` to an exact tag, then download versioned release assets
- Verify checksums, signatures, and attestations before placing binaries on disk
- Keep Linux/macOS and Windows instructions aligned on the same trust story

```sh
VERSION="$(gh release view --json tagName -q .tagName)"
```
