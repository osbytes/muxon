# Agent instructions

This repository is the **muxon** monorepo. The open-source harness is the Go module under `harness/` (`muxon` CLI). Before implementing behavior, read the matching doc under `docs/harness-architecture/` and the [implementation plan](docs/harness-architecture/implementation-plan.md).

## Go code

Follow project rules in `.agents/rules/`:

| Rule | Topic |
|------|--------|
| `muxon-context.mdc` | Layout, deps, specs (always on) |
| `go-format-and-naming.mdc` | Naming, packages, receivers |
| `go-errors-context-concurrency.mdc` | Errors, `context`, goroutines, slog |
| `go-documentation.mdc` | Godoc only — no inline narration |
| `go-project-structure.mdc` | `internal/` boundaries, interfaces |
| `go-cli-muxon.mdc` | Thin `cmd/muxon` + Cobra |
| `go-testing.mdc` | Tests, `testdata`, replay |

## Agent skills (`.agents/skills/`)

Tier 1 skills are vendored in-repo. Invoke with `/skill-name` or let Agent attach them when relevant. See [`.agents/skills/README.md`](.agents/skills/README.md) for the list and update commands.

| Source | Skills |
|--------|--------|
| [spf13/go-skills](https://github.com/spf13/go-skills) | `cobra-viper`, `go`, `go-spec-reviewer`, `go-release` |
| [cxuu/golang-skills](https://github.com/cxuu/golang-skills) | `go-code-review`, `go-linting`, `go-context`, `go-concurrency`, `go-error-handling`, `go-logging`, `go-testing` |

If a skill conflicts with `.agents/rules/` or `docs/harness-architecture/`, **follow the project rules and specs**.

External references: [Effective Go](https://go.dev/doc/effective_go), [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments), [Go module layout](https://go.dev/doc/modules/layout).

## Tool config layout

`.agents/` is the single source of truth. Each tool's directory symlinks back to it:

| Path | Target |
|------|--------|
| `.cursor/rules`, `.claude/rules`, `.opencode/rules` | `.agents/rules` |
| `.cursor/skills`, `.claude/skills`, `.opencode/skills` | `.agents/skills` |
| `AGENT.md` | `AGENTS.md` |

`CLAUDE.md` imports this file. Edit `.agents/` and `AGENTS.md`, never the links. On Windows, enable Developer Mode and run `git config core.symlinks true` before cloning so the links check out as real symlinks.
