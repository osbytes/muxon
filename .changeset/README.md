# Changesets

Every pull request carries a changeset: a small markdown file here that names the bump and says,
for someone using `muxon`, what changed.

```bash
pnpm changeset          # pick muxon, pick patch or minor, write the note
pnpm changeset --empty  # docs-only or internal change
```

CI refuses a PR without one. The "Release" pull request turns the pending changesets into
`CHANGELOG.md` and the next version. See [docs/RELEASING.md](../docs/RELEASING.md).
