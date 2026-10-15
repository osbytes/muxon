# Harness implementation plan

**Scope:** Open-source `thee` CLI and libraries only. The closed [API](../api/README.md) is a separate Go module (`api/`) in this monorepo and is not part of this plan.

**Code root:** Step **Files** paths are relative to the `harness/` module (`harness/cmd/thee/...`, `harness/internal/...`).

**Spec:** Behavior is defined in this directory ([README](./README.md)). This document is the **build order**, package layout, milestones, and v1 cut line. When a step says “per spec”, the linked section wins over this plan.

**North star:** Headless `thee run` fixes a failing unit test in a fixture repo using a **small-model profile** (`two_step`, `compact` prompt), **verify-gated** `finish`, JSONL trace, exit code `0`. TUI and eval reuse the same `internal/` loop.

**Two pillars** (v1, not bolted on later): [smart context](./context-engine.md#smart-context-vs-context-dump) and [reliable tool calling](./tools/README.md#tool-calling-on-small-models). See [README § Small-model strategy](./README.md#small-model-strategy-two-pillars).

---

## How to read this plan

Work proceeds in **ten parts**, in order. **Part 1 is a thin end-to-end loop** (every layer exists, minimally). Parts 2–10 deepen that loop. Do not build large subsystems in isolation and wire them at the end.

- Each **step** is one PR (roughly 1–3 implementation files + tests).
- Steps are labeled `P.N` (part P, step N). Do not start a step until its **Depends on** steps pass `go test ./...`.
- PR title: `[P.N] short description`.

**Milestones:** **M0** (Part 1) scripted loop · **M1** (Part 4) live small model · **A** (Part 7) headless CLI · **B** (Part 8) TUI · **C** (Part 9) eval matrix · **D** (Part 10) release.

---

## Rules for the implementing agent

1. Read only the **Read** links on the current step plus [§ Contracts](#contracts) below.
2. Create only the **Files** listed; extra files should be tests or `testdata` for that step.
3. Do not invent behavior; spec wins. If stuck, pick the simplest behavior that satisfies **Tests** and note it in the PR.
4. Tests: `t.TempDir()`, `t.Setenv("THEE_HOME", ...)`, `testutil.CopyFixture`; never mutate `testdata/fixtures` in place; no network in unit tests.
5. Before PR: `gofmt`, `go vet ./...`, `go test ./...`, `golangci-lint run`.
6. Prompt changes under `internal/assets/prompts/` require golden/`prefix_hash` updates in the same PR.
7. Allowed deps only (see [§ Contracts](#contracts)); no vendor LLM SDKs, testify, or heavy frameworks.
8. `cmd/thee` wires flags and exit codes only; no HTTP, tools, or prompt assembly in `cmd/`.

### Step template

```text
### Step P.N — Title
Depends on: …
Read: …
Files: …
Build: …
Tests: …
Done when: …
```

---

## Binding decisions (v1)

| ID | Decision |
|----|----------|
| D1 | Module `github.com/<org>/thee-harness`; human supplies `<org>` in step 1.1. |
| D2 | `go:embed` only in `internal/assets/` (models.yaml, prompts). |
| D3 | Package `internal/contextengine` (not `context`). |
| D4 | **No OS sandbox in v1.** `shell`/`process`/`browser`/`mcp`/`subagent` return “not available in this build”. Verify/`test` trust-gated. argv-only exec, env scrub, process groups, path checks always on. |
| D5 | `workspace.Changed()` tracks files the harness touched vs originals (not full tree hash). |
| D6 | `models.yaml` includes `roles: { worker: { model_id: … }, … }`. |
| D7 | History: one canonical tool call per action; non-native protocols store assistant text as JSON or fenced block; results as `<tool_result>` user messages. |
| D8 | `grammar` structured output → treat as `json_object` until Part 9. |
| D9 | Patch ladder in Go (udiff + search_replace); no `git` binary required for apply. |
| D10 | Compaction v1: elision + truncate only (ADR-003 open; no summarizer until eval proves need). |
| D11 | No `planning` state in v1; registry still has planning-phase tools for later. |
| D12 | Repo-map v1: in-memory Go symbols + path list; cache `.thee/index/repomap.json`. SQLite optional Part 9. |
| D13 | Classification: router JSON when `router` role set; else `change` + goal-text tags. Finish gate uses workspace changed, not label. |
| D14 | `THEE_HOME` (default `~/.thee`); tests always set it. |
| D15 | HTTP retry: in-house, 3× backoff from 1s. |
| D16 | Director tests use `provider/scripted` until Part 9 `replay` traces exist. |
| D17 | **Salvage lives in `internal/toolcall`**, not `tools/runtime`. Runtime only executes validated `tools.Call`. |
| D18 | Early replay tests use **hand-written** scripted sequences; golden traces recorded after M0. |

---

## Milestones

| ID | Criteria | Part |
|----|----------|------|
| **M0** | `thee run -p "fix the failing test" --trust` on `testdata/fixtures/go-failing-test` passes with scripted provider; trace has `verify.run` pass + `session.end` `done`. | 1 |
| **M1** | Same task live on `two_step`/`compact` (≤~30B); record `malformed_call_rate`, `salvage_rate`, `model_calls_per_tool`. | 4 |
| **A** | Full `thee run` flags, exit 0–3/130, `doctor`, `cost`, `trust`, `eval run`. | 7 |
| **B** | TUI completes same fixture. | 8 |
| **C** | Eval basket × model matrix per [benchmarks](../benchmarks.md). | 9 |
| **D** | Goreleaser, auth, docs, no P0 spec gaps. | 10 |

Stop and read traces if **M1** fails badly before building Parts 5–10.

---

## Out of scope (v1)

Hosted API (optional provider only) · full `serve` · MCP/browser/subagent/process/shell tools · LSP · memory/skills/hooks · planning mode · native `anthropic_messages`/`openai_responses` (optional late Part 9) · embeddings/RAG · compaction summarizer/validate (ADR-003 stage 3–4) · provider failover · team audit log.

---

## Contracts

### Module and dependencies

- Go **1.24+**; binary `thee` from `cmd/thee`.
- Allowed: `cobra`, `yaml.v3`, `BurntSushi/toml`, `go-cmp`, `go-gitignore`, `bubbletea`/`lipgloss`/`bubbles`, `go-keyring`, optional `tiktoken-go`/`modernc.org/sqlite` in Part 9.
- Stdlib first: `context`, `net/http`, `encoding/json`, `log/slog`, `embed`, `os/exec`.

### Repository layout

```text
cmd/thee/
internal/
  assets/          # embed models.yaml + prompts/
  config/          # THEE_HOME, manifest, catalog, toml, credentials
  observability/   # JSONL trace, event types
  provider/        # Request/Response; openai/, scripted/, replay/
  security/        # deny/ignore, redaction, tiers, trust, env scrub
  workspace/       # resolve paths, atomic write, read hash, change tracker, shadow refs
  subproc/         # argv exec, timeouts, process groups
  tools/           # Registry, schema/, impl/
  toolcall/        # native/two_step/fenced, constraint, salvage, repair
  index/           # repo-map
  contextengine/   # BuildTurn, compaction, prefix_hash
  verifier/        # resolve, subprocess, parsers, baseline, gaming
  router/          # roles, escalation, cost, classify
  director/        # state machine, caps, loops, checkpoints
  ui/headless/, ui/tui/
  eval/, e2e/, testutil/
testdata/fixtures/, testdata/traces/, testdata/provider/
```

**Import rule:** `cmd/thee` → `internal/*` (not `e2e`/`testutil`). `director` may import all core packages except `ui/*`. `toolcall` imports `provider`, `tools`, `config`; not `director`.

### On-disk

| Path | Purpose |
|------|---------|
| `$THEE_HOME/config.toml`, `catalog.json`, `trust.json` | User config |
| `<ws>/.thee/sessions/<id>/trace.jsonl` | Trace |
| `<ws>/.thee/sessions/<id>/verify-<n>.log` | Verify log |
| `refs/thee/sessions/<id>/base`, `cp-<n>` | Shadow git refs (user index untouched) |

Session id: `YYYYMMDDTHHMMSSZ-xxxxxx`. `.thee/` excluded from tools; append to `.git/info/exclude` on first session if needed.

### Core interfaces (names are stable)

- **`provider.Provider`:** `Complete` / `Stream`; normalized `Message`, `ToolCall`, `Usage`, `Constraint`, `APIError` kinds (`tools_rejected`, `constraint_rejected`, `context_length`, …).
- **`tools.Tool`:** `Spec()`, `Run(ctx, *Env, args json.RawMessage) (Result, error)`; bad model input → `Result{OK:false, Text:"one line"}`.
- **`tools.Registry`:** `Register`, `Active(tier, phase)`, `Check`, `Schema(name, editFormat)`.
- **`toolcall.Driver`:** `Next(ctx, provider, baseRequest, activeToolNames) (Outcome, …)`; salvage + repair here.
- **`contextengine.BuildTurn`:** returns `messages`, optional `tools[]` (native only), `prefix_hash`, `estimated_input_tokens`.
- **`verifier`:** `ResolveCommand`, `Run(ctx, cmd argv)`, `ParseGoTest` (expand in Part 5), baseline cache.
- **`director.Run`:** returns `Outcome` + `ExitCode` per [session-director § Exit codes](./session-director.md#exit-codes); setup errors → exit 3.

### Headless exit codes

| Code | Meaning |
|------|---------|
| 0 | `done` |
| 1 | `failed` (verify/budget/cap) |
| 2 | `approval_required` |
| 3 | setup (`no_verify_command`, `workspace_untrusted`, bad config, missing key) |
| 130 | `cancelled` |

---

## Part 1 — Walking skeleton (M0)

Goal: fake or real provider → tool call → patch → `finish` → harness `go test ./...` → `done`.

### Step 1.1 — Bootstrap

Depends on: —
Read: [tech-stack](./tech-stack.md), ADR-001/002  
Files: `go.mod`, `cmd/thee/main.go`, `cmd/thee/version.go`, `.github/workflows/ci.yml`, `.golangci.yml`  
Build: Cobra root; `thee version`; CI `go test`, `vet`, `lint`, `build ./...` on linux+darwin; install `rg` in CI.  
Tests: `go test ./cmd/thee/...`  
Done when: green CI on empty packages.

### Step 1.2 — Embedded manifest

Depends on: 1.1  
Read: [providers § Manifest](./providers.md#manifest-modelsyaml)  
Files: `internal/assets/models.yaml`, `internal/assets/embed.go`, `internal/config/manifest.go`  
Build: Embed yaml; parse `providers`, `models`, `roles`; `ApplyDefaults` on capabilities (32k, `two_step`, `compact`, `search_replace`, …).  
Tests: load embedded manifest; worker role resolves.  
Done when: unit test loads manifest.

### Step 1.3 — THEE_HOME and credentials

Depends on: 1.2  
Read: [providers § Credential resolution](./providers.md#credential-resolution-order)  
Files: `internal/config/paths.go`, `internal/config/credential.go`  
Build: `THEE_HOME`; resolve `api_key_env` → env (never log value).  
Tests: missing key error; present key found.  
Done when: credential tests pass.

### Step 1.4 — Observability writer

Depends on: 1.1  
Read: [observability § Event types](./observability.md#event-types)  
Files: `internal/observability/writer.go`, `events.go` (subset: `session.start`, `session.end`, `state.transition`, `tool.call`, `tool.result`, `verify.run`)  
Build: append JSONL; `SchemaVersion` `1.0.0`.  
Tests: two events round-trip.  
Done when: writer tests pass.

### Step 1.5 — Provider types + scripted

Depends on: 1.1  
Read: [providers § Adapters](./providers.md#adapters)  
Files: `internal/provider/types.go`, `internal/provider/scripted/scripted.go`  
Build: `Provider` interface; scripted returns queued `Response`s.  
Tests: two-call queue drains in order.  
Done when: scripted tests pass.

### Step 1.6 — openai_chat adapter

Depends on: 1.5  
Read: [providers § Streaming, Usage](./providers.md#streaming)  
Files: `internal/provider/openai/client.go`, `stream.go`, `testdata/provider/openai_sse_*.txt`  
Build: POST chat/completions; SSE; normalize usage; context cancel; retry helper (D15).  
Tests: `httptest` recorded SSE → expected `Response`.  
Done when: adapter tests pass (no live network).

### Step 1.7 — Workspace open + atomic write

Depends on: 1.1  
Read: [tool-runtime § Files](./tool-runtime.md#files)  
Files: `internal/workspace/workspace.go`, `write.go`  
Build: `Open`, `Resolve`, `WriteFile` (temp+rename), change tracker (D5).  
Tests: reject `../`; atomic write visible.  
Done when: workspace tests pass.

### Step 1.8 — Tool registry + read

Depends on: 1.7  
Read: [tools/read](./tools/read.md)  
Files: `internal/tools/registry.go`, `internal/tools/impl/read.go`  
Build: `Registry`; read line range; cap 400 lines.  
Tests: read fixture file slice.  
Done when: read tool test passes.

### Step 1.9 — write + patch (exact ladder only)

Depends on: 1.8  
Read: [patch](./tools/patch.md), ADR-004  
Files: `internal/tools/impl/write.go`, `patch.go`, `internal/tools/patch/apply.go`  
Build: write 32KiB cap; patch udiff exact + search_replace exact only (D9).  
Tests: table hunks in `testdata/patch/`.  
Done when: patch applies on copied fixture.

### Step 1.10 — Fixture repo

Depends on: 1.9  
Read: [benchmarks § Basket](../benchmarks.md)  
Files: `testdata/fixtures/go-failing-test/**`, `internal/testutil/fixture.go`  
Build: tiny Go module; one failing test; fix is one-line logic bug.  
Tests: `go test` fails before fix, passes after manual fix.  
Done when: fixture documents expected fix.

### Step 1.11 — finish tool + verifier subprocess

Depends on: 1.7, 1.4  
Read: [finish](./tools/finish.md), [verifier](./verifier.md), ADR-007  
Files: `internal/tools/impl/finish.go`, `internal/verifier/verifier.go`, `internal/verifier/resolve.go`  
Build: `finish` sets pending verify; harness runs `go test ./...` argv (trust `--trust` only in skeleton); pass/fail bundle (short excerpt).  
Tests: verifier pass/fail on fixture.  
Done when: finish+verify integration test in temp dir.

### Step 1.12 — Minimal context + native toolcall path

Depends on: 1.5, 1.8  
Read: [context-engine § tools on wire](./context-engine.md#tools-on-the-wire)  
Files: `internal/contextengine/minimal.go`, `internal/toolcall/native.go`  
Build: system + goal + history; native `tools[]` constant; parse single tool call.  
Tests: messages include tool schemas when `native`.  
Done when: build turn test passes.

### Step 1.13 — Minimal director loop

Depends on: 1.11, 1.12, 1.4  
Read: [session-director § State machine](./session-director.md#state-machine) (subset)  
Files: `internal/director/director.go`, `loop.go`  
Build: states `acting` → `tooling` → `verifying` → `done`; max rounds cap; wired scripted provider script: search/read/patch/finish sequence for fixture.  
Tests: scripted run → `done`, workspace fixed.  
Done when: director test on copied fixture.

### Step 1.14 — thee run

Depends on: 1.13, 1.3  
Read: [cli § thee run](./cli.md#thee-run-flags-headless)  
Files: `cmd/thee/run.go`  
Build: `-p`, `--trust`, `--directory`; map `Outcome` to exit codes.  
Tests: `e2e` or `os/exec` run binary with scripted provider env hook (inject scripted in test only).  
Done when: **M0** e2e green in CI.

### Step 1.15 — Live smoke (integration tag)

Depends on: 1.14  
Read: providers  
Files: `internal/e2e/live_test.go` with `//go:build integration`  
Build: optional real `openai_chat` run; record outcome in PR/issue, not CI required.  
Done when: documented one manual run attempted.

---

## Part 2 — Workspace tools and safety

### Step 2.1 — Secret deny + gitignore

Depends on: 1.7  
Read: [security § Ignore rules](./security-and-policy.md#ignore-rules-and-secret-paths)  
Files: `internal/security/policy.go`, `internal/security/ignore.go`  
Build: deny list; `.theeignore`; bulk dirs.  
Tests: `.env` read denied; `node_modules` collapsed in list.

### Step 2.2 — Result caps + redaction + wrappers

Depends on: 2.1, 1.8  
Read: [context-engine § Untrusted](./context-engine.md#untrusted-content), [tool-runtime § Results](./tool-runtime.md#results)  
Files: `internal/security/redact.go`, `internal/tools/result.go`  
Build: wrap `<tool_result trust=…>`; cap bytes/lines.  
Tests: redact API key pattern.

### Step 2.3 — search + list

Depends on: 2.1  
Read: [search](./tools/search.md), [list](./tools/list.md)  
Files: `internal/tools/impl/search.go`, `list.go`  
Build: `rg` wrapper; 50 hits; list 200 entries.  
Tests: find symbol in fixture.

### Step 2.4 — delete

Depends on: 1.7  
Read: [delete](./tools/delete.md)  
Files: `internal/tools/impl/delete.go`  
Done when: delete/move tests pass.

### Step 2.5 — Full patch ladder + stale read

Depends on: 1.9, 1.8  
Read: [patch](./tools/patch.md), [tool-runtime § Stale reads](./tool-runtime.md#files)  
Files: `internal/tools/patch/ladder.go`  
Build: steps 2–4 whitespace, fuzzy, reject; `CheckFresh` before patch/write.  
Tests: table-driven ladder; stale read error message.

### Step 2.6 — Phases + tier checks

Depends on: 1.8  
Read: [tools README § phases](./tools/README.md#tool-calling-on-small-models)  
Files: `internal/tools/phases.go`  
Build: planning vs editing active sets; `Registry.Check`.  
Tests: `patch` rejected in planning phase.

### Step 2.7 — subproc argv-only

Depends on: 2.1  
Read: [tool-runtime § Subprocesses](./tool-runtime.md#subprocesses)  
Files: `internal/subproc/run.go`  
Build: no `sh -c`; env scrub; process group; timeout.  
Tests: timeout kills tree.

---

## Part 3 — Context engine and repo-map

### Step 3.1 — Prompt templates (compact)

Depends on: 1.12  
Read: [context-engine § Prompt variants](./context-engine.md#prompt-variants)  
Files: `internal/assets/prompts/compact/*.tmpl`, `internal/contextengine/render.go`  
Build: ~10 imperative rules; tool catalog one-liners; one worked example for two_step.  
Tests: golden file hash.

### Step 3.2 — standard variant

Depends on: 3.1  
Files: `internal/assets/prompts/standard/*.tmpl`  
Done when: golden updated.

### Step 3.3 — Token estimate (approx + correction ratio)

Depends on: 3.1  
Read: [context-engine § Token estimation](./context-engine.md#token-estimation)  
Files: `internal/contextengine/tokens.go`  
Tests: ratio adjusts after fake usage.

### Step 3.4 — AGENTS.md load

Depends on: 3.1  
Read: [context-engine § Project instructions](./context-engine.md#project-instructions)  
Files: `internal/contextengine/instructions.go`  
Tests: precedence AGENTS.md > CONVENTIONS.md.

### Step 3.5 — Repo-map (in-memory, D12)

Depends on: 2.1  
Read: [repo-map](./tools/repo-map.md), [context-engine § Ranking](./context-engine.md#retrieval-pipeline)  
Files: `internal/index/repomap.go`, `internal/tools/impl/repomap.go`  
Build: Go symbols via `go/parser`; rank by goal overlap; cache JSON.  
Tests: test file ranks high when goal mentions test name.

### Step 3.6 — 12-section layout + cache boundary

Depends on: 3.1–3.5  
Read: [context-engine § Prompt layout](./context-engine.md#prompt-layout-cache-friendly)  
Files: `internal/contextengine/build.go`  
Build: sections 1–12; `prefix_hash` stable when prefix unchanged; phase line in tail only.  
Tests: same session → same `prefix_hash`; phase change does not change hash.

### Step 3.7 — Compaction elision + truncate (D10)

Depends on: 3.6  
Read: [context-engine § Compaction](./context-engine.md#compaction-pipeline)  
Files: `internal/contextengine/compact.go`  
Tests: token estimate drops; verify failure never elided.

### Step 3.8 — Small-window shares

Depends on: 3.6  
Read: [context-engine § Small windows](./context-engine.md#small-windows)  
Tests: 32k budget trims repo-map before AGENTS.md.

---

## Part 4 — Tool-calling stack (M1)

### Step 4.1 — Reasoning separation

Depends on: 1.6  
Read: [providers § Reasoning](./providers.md#reasoning-content)  
Files: `internal/toolcall/reasoning.go`  
Tests: strip `<think>` before parse.

### Step 4.2 — JSON schema from structs

Depends on: 1.8  
Files: `internal/tools/schema/schema.go`  
Tests: read schema has required path fields.

### Step 4.3 — Salvage (D17)

Depends on: 4.2  
Read: [tools § Salvage](./tools/README.md#salvage-before-repair)  
Files: `internal/toolcall/salvage.go`  
Tests: fenced JSON + trailing comma → valid call.

### Step 4.4 — Constraint builder + downgrade

Depends on: 4.2, 1.6  
Read: [tools § Constrained decoding](./tools/README.md#constrained-decoding)  
Files: `internal/toolcall/constraint.go`  
Tests: httptest 400 on schema → retry with `json_object`.

### Step 4.5 — two_step driver

Depends on: 4.3, 4.4, 3.6  
Files: `internal/toolcall/two_step.go`  
Build: step1 enum tool names; step2 flat schema in tail (section 11).  
Tests: scripted two HTTP turns → one `tools.Call`.

### Step 4.6 — fenced driver

Depends on: 4.5  
Files: `internal/toolcall/fenced.go`  
Tests: malformed fence → salvage → execute.

### Step 4.7 — Repair once + director hook

Depends on: 4.5, 1.13  
Read: [tools § Repair](./tools/README.md#repair-and-escalation)  
Files: `internal/director/repair.go`  
Tests: second parse fail returns error to model once.

### Step 4.8 — Wire toolcall into director

Depends on: 4.5–4.7  
Files: `internal/director/director.go`  
Tests: replay malformed → salvaged → `tool.result` ok.

### Step 4.9 — Truncation / served_context flag

Depends on: 1.6, 3.3  
Read: [providers § Local servers](./providers.md#local-and-self-hosted-servers)  
Files: `internal/provider/openai/truncation.go`  
Tests: usage << estimate sets `truncated_by_server` event.

### Step 4.10 — Live small-model smoke (integration)

Depends on: 4.8  
Done when: **M1** recorded (pass or fail with trace).

---

## Part 5 — Director, verifier, trust

### Step 5.1 — Full state machine

Depends on: 4.8  
Read: [session-director § State machine](./session-director.md#state-machine)  
Files: `internal/director/states.go`  
Build: `blocked`, `escalating`, `failed`, `cancelled`; emit all transitions.

### Step 5.2 — Caps and budgets

Depends on: 5.1  
Read: [session-director § Caps](./session-director.md#caps-and-budgets)  
Files: `internal/director/budget.go`  
Tests: `tool_rounds_exhausted`, `budget_exhausted`.

### Step 5.3 — Trust store

Depends on: 2.1  
Read: [security § Workspace trust](./security-and-policy.md#workspace-trust)  
Files: `internal/security/trust.go`, `cmd/thee/trust.go`  
Tests: untrusted headless verify → exit 3.

### Step 5.4 — config.toml / thee.toml precedence

Depends on: 1.3  
Read: [security § Config precedence](./security-and-policy.md#config-precedence)  
Files: `internal/config/config.go`  
Tests: project cannot loosen `max_permission`.

### Step 5.5 — Verify resolution order

Depends on: 1.11  
Read: [verifier § Command resolution](./verifier.md#command-resolution)  
Files: `internal/verifier/resolve.go` (extend)  
Tests: Go module → `go test ./...`.

### Step 5.6 — Go test failure parser + signature

Depends on: 5.5  
Read: [verifier § Failure bundle](./verifier.md#failure-bundle-to-model)  
Files: `internal/verifier/parse_go.go`  
Tests: sample output → `failures[]` + signature for loop detector.

### Step 5.7 — Baseline parallel to first turn

Depends on: 5.5  
Read: [verifier § Baseline](./verifier.md#baseline)  
Tests: baseline fail message in first tool context.

### Step 5.8 — Verify gaming paths

Depends on: 5.5  
Read: [verifier § Verify gaming](./verifier.md#verify-gaming)  
Files: `internal/verifier/gaming.go`  
Tests: `*_test.go` edit flagged.

### Step 5.9 — Session base shadow ref

Depends on: 1.7  
Read: [session-director § Workspace state](./session-director.md#workspace-state-at-start), [checkpoint](./tools/checkpoint.md)  
Files: `internal/workspace/snapshot.go`  
Build: dirty tree → `refs/thee/sessions/<id>/base` without touching user index.

### Step 5.10 — Checkpoints + undo

Depends on: 5.9, 1.13  
Files: `internal/director/checkpoint.go`, `internal/tools/impl/checkpoint.go` (minimal)  
Tests: restore after bad patch.

### Step 5.11 — Loop detection

Depends on: 5.1  
Read: [session-director § Loop detection](./session-director.md#loop-detection)  
Files: `internal/director/loop.go`  
Tests: repeated read → nudge event.

### Step 5.12 — todo + ask_user

Depends on: 5.1  
Read: [todo](./tools/todo.md), [ask_user](./tools/ask_user.md)  
Files: `internal/tools/impl/todo.go`, `ask_user.go`  
Tests: open todo blocks finish; headless ask_user continues.

### Step 5.13 — Director replay integration test

Depends on: 5.1–5.11  
Files: `internal/director/director_replay_test.go`  
Build: scripted trace: verify fail → patch → verify pass → done.  
Done when: test green before Part 8 TUI.

---

## Part 6 — Router, escalation, cost

### Step 6.1 — catalog.json merge

Depends on: 1.2  
Read: [providers § catalog](./providers.md#user-provider-registry-catalogjson)  
Files: `internal/config/catalog.go`  
Tests: user provider overrides base_url.

### Step 6.2 — Role binding + provider factory

Depends on: 6.1, 1.6  
Read: [model-router § Roles](./model-router.md#roles)  
Files: `internal/router/bind.go`, `internal/router/provider.go`  
Tests: worker resolves to openai client.

### Step 6.3 — Cost accounting

Depends on: 6.2  
Read: [model-router § Cost](./model-router.md#cost-accounting)  
Files: `internal/router/cost.go`  
Tests: cached tokens priced separately.

### Step 6.4 — Escalation triggers

Depends on: 5.2, 6.2  
Read: [model-router § Escalation](./model-router.md#escalation-triggers)  
Files: `internal/router/escalate.go`  
Tests: N verify fails → escalate event (mock).

### Step 6.5 — Classification (D13)

Depends on: 6.2, 4.4  
Files: `internal/router/classify.go`  
Tests: question goal → `task_type` question; patch still requires verify if changed.

---

## Part 7 — Headless CLI (Milestone A)

### Step 7.1 — All run flags

Depends on: 5.x, 6.x  
Read: [cli § run flags](./cli.md#thee-run-flags-headless)  
Files: `cmd/thee/run.go`  
Build: `--verify-cmd`, `--permission`, `--yolo`, budgets, `--model`, `--no-plan`, `--eval`.

### Step 7.2 — Presenters (--json, -q)

Depends on: 7.1  
Files: `internal/ui/headless/present.go`  
Tests: NDJSON includes `session.end`.

### Step 7.3 — SIGINT → 130

Depends on: 7.1  
Read: [session-director § Interrupts](./session-director.md#interrupts-and-steering)  
Tests: cancel mid-run.

### Step 7.4 — doctor

Depends on: 6.1, 2.3  
Read: [cli § doctor](./cli.md#first-run-and-doctor)  
Files: `cmd/thee/doctor.go`  
Tests: missing key reported, not printed.

### Step 7.5 — cost

Depends on: 1.4  
Files: `cmd/thee/cost.go`  
Tests: read last trace totals.

### Step 7.6 — trust subcommands

Depends on: 5.3  
Files: `cmd/thee/trust.go` (complete)

### Step 7.7 — eval run

Depends on: 7.1  
Read: [benchmarks](../benchmarks.md), [observability § Eval](./observability.md#eval-integration)  
Files: `internal/eval/task.go`, `cmd/thee/eval.go`, `testdata/eval/fix-test.json`  
Done when: **Milestone A** documented in repo README.

---

## Part 8 — Interactive TUI (Milestone B)

### Step 8.1 — Bubble Tea shell

Depends on: 5.13  
Read: [cli § wireframe](./cli.md#interactive-layout-wireframe)  
Files: `internal/ui/tui/app.go`, `cmd/thee/chat.go`  
Build: header, transcript, status, input regions.

### Step 8.2 — Streaming + tool lines

Depends on: 8.1  
Files: `internal/ui/tui/stream.go`  
Tests: model test with fake stream.

### Step 8.3 — Approval modals + trust first-run

Depends on: 8.1, 5.3  
Files: `internal/ui/tui/modal.go`

### Step 8.4 — Interrupts + slash commands

Depends on: 8.1  
Read: [cli § Slash](./cli.md#slash-commands-interactive)  
Files: `internal/ui/tui/input.go`  
Build: `/help`, `/abort`, `/undo`, `/escalate`, `/cost`.

### Step 8.5 — Multi-goal in one session

Depends on: 8.1  
Read: [session-director § terminal states](./session-director.md#state-machine)  
Done when: **Milestone B** same fixture as A.

---

## Part 9 — Probe, eval, hardening (Milestone C)

### Step 9.1 — trace record / replay provider

Depends on: 5.13  
Read: [observability § Trace replay](./observability.md#trace-replay-harness-self-tests)  
Files: `internal/provider/replay/replay.go`, `cmd/thee/trace.go`  
Tests: hash mismatch fails with diff (D18).

### Step 9.2 — models list / show

Depends on: 6.1  
Files: `cmd/thee/models.go`

### Step 9.3 — models probe (thresholds TBD in doc only)

Depends on: 9.2, 4.x  
Read: [model-router § Probe](./model-router.md#probe)  
Files: `internal/router/probe.go`  
Build: served_context + protocol parse rates; write catalog snippet.

### Step 9.4 — test + git tools

Depends on: 2.7, 5.3  
Read: [test](./tools/test.md), [git](./tools/git.md)  
Files: `internal/tools/impl/test.go`, `git.go`

### Step 9.5 — Eval basket × matrix

Depends on: 7.7  
Files: `testdata/eval/*.json`, nightly workflow optional  
Done when: **Milestone C** `results.jsonl` + report stub.

### Step 9.6 — resume

Depends on: 5.10, 8.1  
Files: `cmd/thee/resume.go`

### Step 9.7 — Optional anthropic adapter

Depends on: 1.5  
Read: ADR-011  
Files: `internal/provider/anthropic/` (if time).

---

## Part 10 — Release (Milestone D)

### Step 10.1 — auth login (keyring)

Depends on: 1.3  
Files: `cmd/thee/auth.go`

### Step 10.2 — trace export

Depends on: 9.1  
Files: `cmd/thee/trace.go` export subcommand

### Step 10.3 — gc, completion

Depends on: 5.10  
Files: `cmd/thee/gc.go`, `completion.go`

### Step 10.4 — serve stub

Depends on: 1.1  
Files: `cmd/thee/serve.go` → exit 3 message.

### Step 10.5 — goreleaser + README + adrs checklist

Depends on: all  
Files: `.goreleaser.yml`, root README run instructions  
Done when: **Milestone D** binaries on tag; [adrs § checklist](./adrs.md#spec-checklist) P0 closed.

---

## Prerequisites (spec prose)

| Item | Doc | Blocks |
|------|-----|--------|
| `compact` / `standard` prompts | [context-engine](./context-engine.md#prompt-variants) | Part 3 |
| `models.yaml` + `roles` | [providers](./providers.md#manifest-modelsyaml) | Part 1 |
| Probe thresholds | [model-router](./model-router.md#probe) | Step 9.3 only |
| Eval fixtures | [benchmarks](../benchmarks.md) | Part 7/9 |

---

## Testing strategy (continuous)

| Layer | Method |
|-------|--------|
| Director + caps | scripted → replay (Part 9) |
| Prompt | `prefix_hash` + goldens |
| Tools | unit + copied fixture |
| Provider | `httptest` SSE |
| Patch | table `testdata/patch/` |
| Integration | `integration` tag + optional DeepInfra nightly |

Add director replay test in **Part 5** before TUI (step 5.13).

---

## Risk register

| Risk | Mitigation |
|------|------------|
| Small model cannot complete loop | M0/M1 live smokes; stop if M1 fails |
| `two_step` latency | `model_calls_per_tool`; probe one-step `oneOf` (9.3) |
| Multi-language index | Go-first (D12); search still works |
| Ollama truncation | `num_ctx`, probe, step 4.9 |
| Scope creep | Out-of-scope table; stub `full` tools |
| Prompt drift | Goldens + replay hash (9.1) |
| ADR-003 | D10: no summarizer until eval says so |

---

## Suggested parallel work (non-code)

While Part 2–3 proceed: draft `compact`/`standard` prose; pin `models.yaml` rates against DeepInfra; expand eval task JSON descriptions.

---

## Related

- [tech-stack.md](./tech-stack.md) — libraries and CI  
- [cli.md](./cli.md) — commands and TUI  
- [adrs.md](./adrs.md) — decisions and spec checklist  
- [README.md](./README.md) — architecture overview  
