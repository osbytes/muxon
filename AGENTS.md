# Agent instructions

This repository is the **thee** monorepo. The open-source harness is the Go module under `harness/` (`thee` CLI). Before implementing behavior, read the matching doc under `docs/harness-architecture/` and the [implementation plan](docs/harness-architecture/implementation-plan.md).

## Go code

Follow project Cursor rules in `.cursor/rules/`:

| Rule | Topic |
|------|--------|
| `thee-harness-context.mdc` | Layout, deps, specs (always on) |
| `go-format-and-naming.mdc` | Naming, packages, receivers |
| `go-errors-context-concurrency.mdc` | Errors, `context`, goroutines, slog |
| `go-documentation.mdc` | Godoc only — no inline narration |
| `go-project-structure.mdc` | `internal/` boundaries, interfaces |
| `go-cli-thee.mdc` | Thin `cmd/thee` + Cobra |
| `go-testing.mdc` | Tests, `testdata`, replay |

## Agent skills (`.agents/skills/`)

Tier 1 skills are vendored in-repo. Invoke with `/skill-name` or let Agent attach them when relevant. See [`.agents/skills/README.md`](.agents/skills/README.md) for the list and update commands.

| Source | Skills |
|--------|--------|
| [spf13/go-skills](https://github.com/spf13/go-skills) | `cobra-viper`, `go`, `go-spec-reviewer`, `go-release` |
| [cxuu/golang-skills](https://github.com/cxuu/golang-skills) | `go-code-review`, `go-linting`, `go-context`, `go-concurrency`, `go-error-handling`, `go-logging`, `go-testing` |

If a skill conflicts with `.cursor/rules/` or `docs/harness-architecture/`, **follow the project rules and specs**.

External references: [Effective Go](https://go.dev/doc/effective_go), [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments), [Go module layout](https://go.dev/doc/modules/layout).
