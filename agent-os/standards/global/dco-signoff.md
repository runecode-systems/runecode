# DCO Sign-off

- Every commit in every PR must include a DCO `Signed-off-by:` line
- Preferred way to sign commits:

```sh
git commit -s
```

Fix missing sign-offs:
- Last commit: `git commit --amend -s`
- Multiple commits: `git rebase --signoff origin/main`

Source of truth: `CONTRIBUTING.md` and `DCO`
