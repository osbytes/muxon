# Releasing

`muxon` is one Go binary (`harness/cmd/muxon`) shipped as archives for Linux, macOS and Windows,
versioned with [changesets](https://github.com/changesets/changesets). The version lives in the
root `package.json`, which exists only for the release tooling; nothing is published to npm
(`"private": true`). goreleaser stamps it into the binary (`muxon version`).

## Every pull request

1. Make the change. Run `go test ./harness/...`, `go vet ./harness/...` and `gofmt`.
2. `pnpm install` once, then `pnpm changeset`: pick `muxon`, pick the bump, write the note **as a
   user reads it** ("`muxon run` accepts `--budget-usd`", not "refactor director loop"). One
   changeset per user-visible change; a PR can have several. Docs-only or internal PRs:
   `pnpm changeset --empty`.
3. If the change needs anything from the user at upgrade time (a moved `~/.muxon` file, a flag that
   changed meaning, a config key renamed, a trace schema bump), add an `**Upgrade:**` line to the
   changeset. It ends up in the changelog and the GitHub release.
4. CI (`ci.yml`) tests and builds on all three platforms, and refuses a PR without a changeset.

## Versions (before 1.0)

- **patch**: fixes, small improvements, dependency updates, anything with no new flag or setting.
- **minor**: new commands, new flags or config keys, changed defaults, changes to session or trace
  files on disk, anything a user would call "new".
- **major**: reserved for 1.0. Until then, breaking changes are minor and say so in `**Upgrade:**`.

## Cutting a release

1. Merging to `main` runs `version.yml`, which keeps a **"Release"** pull request open. It bumps
   the version, writes `CHANGELOG.md` from the pending changesets and deletes them. Review it like
   any PR: is the changelog readable, are upgrade notes present.
2. Merge. `version.yml` runs `pnpm release:tag`, which pushes `vX.Y.Z`. `release.yml` checks the
   tag matches `package.json`, builds the archives with goreleaser (`.goreleaser.yml`), attests
   them (`gh attestation verify <archive> --owner adavila0703`), and publishes the GitHub release
   with the archives and `checksums.txt`. The release notes are the changelog section
   (`scripts/release-notes.mjs`; maintainers' "Thanks @…!" credits are dropped, outside
   contributors' stay).
3. Download one archive and check `muxon version` prints the new version.

A manual run of `release.yml` (`workflow_dispatch`) builds a goreleaser snapshot and publishes
nothing.

By hand, without the workflow: `pnpm release:version` on `main`, commit, `pnpm release:tag`.

## Pre-releases

`pnpm changeset pre enter beta` on `main` (or a release branch); every release is then
`X.Y.Z-beta.N` and goreleaser marks the GitHub release as a pre-release (`prerelease: auto`).
`pnpm changeset pre exit` before the final release.

## Hotfixes

Branch from the tag (`git switch -c hotfix/1.2.1 v1.2.0`), cherry-pick the fix with its
changeset, `pnpm release:version`, commit, `pnpm release:tag`. Merge the branch back into `main`
so the changelog and version stay ahead.

## Setup (once per repository)

Repository settings live outside the code, so this list is the record of them; change it with the
setting.

- `secrets.RELEASE_TOKEN`: a fine-grained PAT of `adavila0703` with `contents: write` and
  `pull-requests: write` on this repository. It is required because tags pushed with the default
  `GITHUB_TOKEN` do not trigger `release.yml`. changesets/action (the Release PR),
  `pnpm release:tag` and the Dependabot changeset job all use it. `release.yml` itself publishes
  with `GITHUB_TOKEN`.
- Settings → Actions → General:
  - "Workflow permissions": **Read repository contents**, and "Allow GitHub Actions to create and
    approve pull requests" **off**. Every workflow declares its own `permissions:`.
- Settings → Branches → `main`: require the `changeset` check and the `test` matrix (`ci`). The
  Release PR is merged like any other.
- Settings → Rules → tag ruleset `v*`: restrict creation, update, deletion and force pushes, with
  only `adavila0703` on the bypass list, so only the release flow creates version tags.
