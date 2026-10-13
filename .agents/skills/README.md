# Project agent skills (Tier 1)

Vendored via [skills CLI](https://github.com/vercel-labs/skills). Cursor discovers these from `.agents/skills/` automatically.

**Precedence:** For thee-harness layout, godoc-only comments, and `internal/` boundaries, **`.cursor/rules/` and this repo’s architecture docs win** if a skill suggests a different structure (e.g. `pkg/`, heavy layering).

## spf13/go-skills

| Skill | Use |
|-------|-----|
| `cobra-viper` | `cmd/thee` — Cobra wiring, config, CLI tests |
| `go` | Idiomatic Go, flat packages, stdlib-first |
| `go-spec-reviewer` | Review design/spec before large implementations |
| `go-release` | GoReleaser, versioning, release hygiene |

Source: https://github.com/spf13/go-skills

## cxuu/golang-skills (subset)

| Skill | Use |
|-------|-----|
| `go-code-review` | Pre-merge diff review + mechanical checks |
| `go-linting` | `golangci-lint` setup and CI |
| `go-context` | `context.Context` propagation |
| `go-concurrency` | Goroutines, channels, errgroup |
| `go-error-handling` | Wrapping, sentinels, error flow |
| `go-logging` | `slog` patterns |
| `go-testing` | Table tests, organization, helpers |

Scripts (e.g. `go-code-review/scripts/pre-review.sh`) live under each skill directory when present.

Source: https://github.com/cxuu/golang-skills

## Refresh

```bash
npx skills add spf13/go-skills --skill cobra-viper --skill go --skill go-spec-reviewer --skill go-release -y
npx skills add cxuu/golang-skills --skill go-code-review --skill go-linting --skill go-context --skill go-concurrency --skill go-error-handling --skill go-logging --skill go-testing -y
```
