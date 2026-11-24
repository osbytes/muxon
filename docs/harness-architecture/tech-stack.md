# Tech stack

**Decision:** The muxon **core** is implemented in **Go** as a single CLI binary (`muxon`). Our hosted API lives in a separate repo; BYOK direct mode does not depend on it.

**Go version:** **1.24+** (pin current stable in `go.mod` at implementation time).

Agent capabilities (search, patch, shell, etc.) are **product architecture**, not stack choices—see [tool-runtime.md](./tool-runtime.md).

---

## Rationale

| Goal | Why Go |
|------|--------|
| CLI-first delivery | Fast iteration on HTTP, config, and local I/O |
| BYOK inference | `net/http` + SSE for OpenAI-compatible APIs |
| Distribution | Single cross-compiled binary; no user runtime |
| Interactive UX | Bubble Tea / Charm ecosystem |
| One codebase | CLI, TUI, and `serve` without a second language |

Rust/Python references (Ante, Aider, Tau) inform design; the harness is **Go only**.

---

## Language & runtime

| Item | Choice |
|------|--------|
| Language | Go |
| Concurrency | `context.Context`, goroutines where streaming/async helps |
| Subprocess / files | `os/exec`, `os`, `path/filepath` (how we invoke external binaries is defined in [tool-runtime.md](./tool-runtime.md)) |
| Embedding defaults | `embed` for shipped `models.yaml` |

---

## Repository layout

```text
muxon/           # monorepo root (docs, go.work)
├── harness/            # OSS muxon CLI module
│   ├── cmd/muxon/
│   ├── internal/
│   ├── testdata/
│   ├── go.mod
│   └── go.sum
├── api/                # closed API module (separate go.mod later)
├── docs/
└── go.work
```

Package names under `internal/` follow [README.md](./README.md) (director, context, router, provider, …). **This doc does not define package behavior**—only that we use a standard Go module layout with `internal/` boundaries so `muxon serve` can reuse them in a long-lived process (ADR-002).

---

## CLI & terminal UI

| Surface | Libraries |
|---------|-----------|
| Commands and headless output | [spf13/cobra](https://github.com/spf13/cobra) or std `flag`; stdout/stderr; optional `--json` |
| Formatting | [charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss) |
| TUI | [charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea) |
| `muxon serve` | JSONL over stdio or WebSocket ([session-director.md](./session-director.md#surfaces)) |

---

## Configuration & serialization

| Data | Format | Library |
|------|--------|---------|
| `muxon.toml` | TOML | [BurntSushi/toml](https://github.com/BurntSushi/toml) or [pelletier/go-toml](https://github.com/pelletier/go-toml) |
| `models.yaml` | YAML | [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) |
| `catalog.json` | JSON | `encoding/json` |
| Secrets | Env / OS keychain | `os.Getenv`, keychain via [zalando/go-keyring](https://github.com/zalando/go-keyring); no secrets in repo |

Config precedence: [security-and-policy.md](./security-and-policy.md#config-precedence). BYOK: [providers.md](./providers.md).

---

## HTTP & inference client

| Concern | Choice |
|---------|--------|
| Client | `net/http` with timeouts; per-provider `http.Client` |
| Streaming | SSE reader (small internal helper or minimal dep) |
| Payloads | `encoding/json` |
| Auth | `Authorization: Bearer` from user-resolved API keys |
| Retries | [cenkalti/backoff](https://github.com/cenkalti/backoff) v4 or small in-house helper for 429s |

Three adapters behind one interface ([providers.md](./providers.md#adapters)): `openai_chat`, `anthropic_messages`, `openai_responses`. Each is a small hand-written REST client over `net/http`, not a vendor SDK, so reasoning fields, cache markers, and usage details are under our control and identical across providers that share a wire format.

| Concern | Choice |
|---------|--------|
| Constrained decoding | JSON Schema generated from the Go tool structs; GBNF grammars generated from the same schemas for llama.cpp-style servers |
| Lenient JSON for salvage | Small in-house repair pass, then `encoding/json` strict decode |
| Tokenizers | `o200k` via [pkoukk/tiktoken-go](https://github.com/pkoukk/tiktoken-go); Hugging Face `tokenizer.json` via [sugarme/tokenizer](https://github.com/sugarme/tokenizer) (pure Go); `approx` fallback |

---

## Observability (implementation)

| Concern | Choice |
|---------|--------|
| Traces | Append-only JSONL via `encoding/json` |
| App logging | [log/slog](https://pkg.go.dev/log/slog) |
| Event schema | [observability.md](./observability.md) |

---

## Testing, lint, CI

| Concern | Choice |
|---------|--------|
| Unit / golden | `testing`, `testdata/` |
| Loop regression | Trace replay: recorded dev-profile traces served by a `replay` provider; asserts state transitions, tool calls, outcome, and prompt hashes ([observability.md](./observability.md#trace-replay-harness-self-tests)) |
| Scripted model | In-process fake provider returning canned tool calls, for unit tests of the director and caps |
| HTTP fakes | `net/http/httptest` |
| Integration | Build tag `integration`; optional live API in CI with secrets |
| Lint | `golangci-lint` |
| Format | `gofmt`, `goimports` |
| CI | GitHub Actions: test, lint, `go build ./...` |

---

## Build & release

| Item | Choice |
|------|--------|
| Module path | TBD (e.g. `github.com/<org>/muxon`) |
| Artifacts | `goreleaser` or manual `GOOS`/`GOARCH` matrix |
| Distribution | GitHub Releases (single `muxon` binary per platform) |

---

## Closed API (separate repo)

Not part of the open-source harness module. Spec: [../api/README.md](../api/README.md).

| Layer | Choice |
|-------|--------|
| Service | Go, private repo |
| Role | OpenAI-compatible proxy to DeepInfra with our key |
| Harness | Public CLI; BYOK still talks to providers directly |

---

## Dependency policy

- **Stdlib first:** `context`, `net/http`, `encoding/json`, `log/slog`, `embed`, `os/exec`.
- **Add deps deliberately:** cobra, yaml, backoff, bubbletea, keyring, tiktoken—one obvious owner package each.
- **Avoid:** ORMs, general-purpose agent frameworks, heavy DI kits.

---

## Related

- [tool-runtime.md](./tool-runtime.md) — agent tools & execution (product architecture)  
- [adrs.md](./adrs.md) — ADR-001 Go, ADR-002 monolith  
- [providers.md](./providers.md) — BYOK  
- [README.md](./README.md) — component overview  
