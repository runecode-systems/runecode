---
schema_version: 1
id: global/source-quality-reviewed-exceptions
title: Source-Quality Reviewed Exceptions
status: active
aliases:
    - agent-os/standards/global/source-quality-reviewed-exceptions
---

# Source-Quality Reviewed Exceptions

- Keep source-quality exceptions in checked-in files:
  - `.source-quality-baseline.json`
  - `.source-quality-config.json`
- Do not rely on reviewer memory or PR comments for active exceptions.
- Each exception should include:
  - exact path or scope
  - allowed limit or override
  - rationale
  - follow-up or review reference
- Rules:
  - prefer narrow exceptions over broad file-wide escapes
  - reduce or remove exceptions over time
  - do not increase limits without explicit review
  - Tier 1 suppressions use reviewed config, not casual inline comments
