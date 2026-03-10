# PR Review Finding Format

For each finding, include:
- Severity: `Critical|High|Medium|Low`
- File path
- Impact (what breaks / risk)
- Concrete fix recommendation

Prioritize: security -> correctness -> reliability -> portability -> maintainability

De-prioritize style-only comments unless they hide risk or violate a documented convention.

```text
Severity: High
File: path/to/file
Impact: ...
Recommendation: ...
```
