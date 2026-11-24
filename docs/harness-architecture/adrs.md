# Architecture decisions

Each decision records what we chose, why, and what it costs. Resolve open ADRs before writing the code they govern.

| ID | Question | Options | Status |
|----|----------|---------|--------|
| **ADR-001** | Implementation language? | Rust · **Go** · Python | **Decided** — Go; see [tech-stack.md](./tech-stack.md) |
| **ADR-002** | Client–daemon split? | **Monolith CLI** · Ante-style daemon + headless client | **Decided** — monolith binary; `serve` reuses the same `internal/` packages |
| **ADR-003** | Compaction strategy? | Elision + truncate only · + cheap summarizer · Cliff-style drop-only | **Open** (lean elision + truncate + optional summarize) |
| **ADR-004** | Edit format? | Unified diff only · SEARCH/REPLACE only · **Per-model format + apply ladder** | **Decided** |
| **ADR-005** | Default verify command? | **Auto-detect with override** · Require `muxon.toml` | **Decided** — auto-detect; strict mode with nothing detected is a setup error, not a weak pass |
| **ADR-006** | Our API vs BYOK? | **BYOK always; our API optional** | **Decided** — BYOK is the required path; our API is never the only option |
| **ADR-007** | Who runs verification? | Model via `shell` · **Harness subprocess** | **Decided** — harness subprocess, trust-gated |
| **ADR-008** | Tool calling protocol? | One protocol for all models · **Per-model capability profile** | **Decided** — `native` / `two_step` / `fenced` from the manifest |
| **ADR-009** | Repo-supplied config and hooks? | Always honor · **Workspace trust gate** | **Decided** — untrusted repos can only tighten policy |
| **ADR-010** | How are tool calls made reliable on small models? | Repair turns only · **Constrain, salvage, then repair** | **Decided** |
| **ADR-011** | Wire APIs? | OpenAI-compatible only · **OpenAI-compatible plus native Anthropic and OpenAI Responses** | **Decided** |
| **ADR-012** | Prior reasoning in history? | Always strip · Always keep · **Per-model `reasoning_history`** | **Decided** |
| **ADR-013** | Stuck-loop handling? | Round cap only · **Pattern detection, nudge, escalate, fail** | **Decided** |
| **ADR-014** | Prompt wording and sampling? | One prompt for all · **Per-model `prompt_variant` and `sampling`** | **Decided** |
| **ADR-015** | Context sizing? | Trust advertised window, fixed prefix sizes · **Measured `served_context`, window-scaled prefix** | **Decided** |

### ADR-001 — Go implementation language

- **Context:** Need a fast headless CLI, BYOK HTTP, subprocess tools, and a TUI; team prefers Go.
- **Decision:** Implement harness in **Go 1.24+**; single `muxon` binary; details in [tech-stack.md](./tech-stack.md).
- **Consequences:** No Rust/Python core; cross-compile releases; Bubble Tea for the TUI.
- **Date:** 2026-10-12

### ADR-002 — Monolith binary

- **Context:** A daemon adds ops complexity before the loop is proven.
- **Decision:** **Monolith** `cmd/muxon`; packages under `internal/` mirror the architecture docs.
- **Consequences:** `muxon serve` reuses the same packages in a long-lived process instead of a separate daemon.
- **Date:** 2026-10-12

### ADR-004 — Edit formats and apply ladder

- **Context:** The original decision was strict unified diff via `git apply`, where a failed hunk stays failed. Small models often emit malformed or slightly-off diffs, and Aider's benchmarks show edit-format fit varies by model. Patch failure is the largest risk to the frugal-model thesis.
- **Decision:** `patch` accepts **unified diff** or **SEARCH/REPLACE**, chosen per model by `edit_format` in its capability profile; the model sees only one schema. Application uses a fixed ladder: exact → whitespace-tolerant → fuzzy locate with a **unique** match only → targeted reject showing the current lines. Ambiguous matches are never applied. Details in [tools/patch.md](./tools/patch.md).
- **Consequences:** Two parsers to maintain. `percent_cases_well_formed` counts only exact and whitespace-tolerant applies, so tolerance cannot hide a model that cannot edit.
- **Date:** 2026-10-12, amended 2026-10-13

### ADR-005 — Verify command resolution

- **Context:** Requiring config adds friction; a silent weak fallback (`git diff --stat`) lets strict mode pass without evidence.
- **Decision:** Auto-detect (npm/pnpm/yarn, make, pytest, go, cargo) with `--verify-cmd` and `[verify] command` overrides. In strict mode, no detected command means interactive asks and headless exits 3 (`no_verify_command`). `git diff --stat` is never a strict pass.
- **Consequences:** Some repos need one line of config to run headless. Details in [verifier.md](./verifier.md#command-resolution).
- **Date:** 2026-10-13

### ADR-007 — Verification runs in the harness

- **Context:** Strict verify needed test runs, but that path went through model `shell`, which is `full` tier and off by default; headless denied it. Small models are also unreliable at shell plus "I'm done" judgment.
- **Decision:** The verifier spawns the verify command itself, with its own subprocess policy (long timeout, full log on disk, small failure bundle to the model). Model `shell` is optional debug only. A `test` tool gives targeted runs through the same command. Execution of repo-supplied commands is gated by workspace trust, not permission tier.
- **Consequences:** The default `edit` tier can verify. There is no "verifying" tool phase. Headless needs `--trust` (or an explicit command) to verify. Details in [verifier.md](./verifier.md).
- **Date:** 2026-10-13

### ADR-008 — Capability-driven tool protocol

- **Context:** The harness must run frontier models and 8B open models alike. A frontier-style loop (all schemas in every prompt, parallel native calls — OpenCode) breaks small models; a small-model loop (one tool per two calls) wastes latency on frontier models. Varying `tools[]` per state also broke prompt caching.
- **Decision:** Each model's capability profile sets `tool_protocol` (`native`, `two_step`, `fenced`), `parallel_tool_calls`, `edit_format`, context window, tokenizer, and cache mode. Unknown models get conservative defaults; `muxon models probe` calibrates. `tools[]` is either the full tier set, constant for the session, or omitted; phase restriction lives in the prompt tail and the runtime.
- **Consequences:** Manifest schema grows ([providers.md](./providers.md#capability-fields)). One loop serves every model class. Observability tracks parse failures and calls per tool per model so profiles can be tuned.
- **Date:** 2026-10-13

### ADR-009 — Workspace trust

- **Context:** `.muxon/hooks/`, project `muxon.toml`, skills, memory, and the verify command are repo-supplied. Without a gate, running `muxon` in a cloned hostile repo is running its scripts.
- **Decision:** Per-workspace trust stored in `~/.muxon/trust.json` with a hash of trust-relevant files. Untrusted: no project hooks, skills, memory, or provider config; project security config may only tighten; verify command needs one approval (headless: `--trust`). Details in [security-and-policy.md](./security-and-policy.md#workspace-trust).
- **Consequences:** One prompt on first open. CI passes `--trust`.
- **Date:** 2026-10-13

### ADR-010 — Constrain, salvage, then repair

- **Context:** Malformed tool calls are the most common way a small model fails a task it could have solved. A repair turn costs a full model call and often fails the same way. Most backends that serve open models can constrain output to a JSON schema or grammar, and most remaining failures are a correct call in the wrong place (inside text, in a fence, with a trailing comma).
- **Decision:** Tool-call intake runs three layers in order. Constrained decoding where `structured_output` allows it, so the call cannot be malformed. A deterministic salvage pass for in-text and slightly broken calls, which must produce a schema-valid call without guessing. One repair turn, then escalation. Constraint also enables a one-step mode for `two_step` models. Details in [tools/README.md](./tools/README.md#constrained-decoding).
- **Consequences:** Two new profile fields (`structured_output`, `text_tool_dialect`) and a schema-to-grammar generator. Salvage can hide a wrong profile, so `salvage_rate` is a first-class metric.
- **Date:** 2026-10-13

### ADR-011 — Native Anthropic and OpenAI Responses adapters

- **Context:** BYOK users bring Anthropic and OpenAI keys, usually for escalation. Their OpenAI-compatible layers drop explicit cache markers, signed thinking, and reasoning items, which costs money (no caching) and breaks multi-turn tool use with reasoning on.
- **Decision:** Provider entries name a wire API: `openai_chat` (default), `anthropic_messages`, or `openai_responses`. All three sit behind one normalized request and response. Gateways that translate formats are configured by what they actually expose. Details in [providers.md](./providers.md#adapters).
- **Consequences:** Three adapters to maintain and test against recorded fixtures. Usage normalization grows a cache-write field.
- **Date:** 2026-10-13

### ADR-012 — Per-model reasoning history

- **Context:** Reasoning comes back as `<think>` text, a `reasoning_content` field, signed thinking blocks, or encrypted items. DeepSeek-style APIs reject prior reasoning; Anthropic requires signed blocks to be returned with their tool results; Qwen recommends stripping. One rule cannot satisfy all of them.
- **Decision:** Reasoning is always separated from the answer before any parser sees it. What is sent back is set by `reasoning_history` (`strip`, `keep`, `keep_signed`). Signed blocks are never edited or summarized. Details in [providers.md](./providers.md#reasoning-content).
- **Consequences:** Compaction must treat signed blocks as atomic. Failover strips reasoning, because it is not portable between providers.
- **Date:** 2026-10-13

### ADR-013 — Loop detection

- **Context:** Small models repeat themselves: the same call, the same read, the same failing patch. A round cap stops this only after most of the budget is gone.
- **Decision:** The director detects repetition patterns from data the trace already records (`args_hash`, file hashes, verify failure signatures). The response is one nudge, then escalation, then `failed` (`loop_detected`). Repeated identical reads are answered from cache. Details in [session-director.md](./session-director.md#loop-detection).
- **Consequences:** A new terminal reason and event type. Thresholds need tuning against the eval basket; too tight and the director interrupts legitimate retries.
- **Date:** 2026-10-13

### ADR-014 — Per-model prompt variant and sampling

- **Context:** The prompt that works for a frontier model (long rules with reasons) is noise to a small one, which follows short imperatives and copies examples. Model families also publish recommended sampling settings, and running a coder model at the wrong temperature visibly hurts tool calls.
- **Decision:** `prompt_variant` (`compact`, `standard`) picks the wording of the system rules, tool catalog, step-2 schema, and repair message. Structure and rules are identical. `sampling` carries per-model decoding settings, shipped with vendor-recommended values. Details in [context-engine.md](./context-engine.md#prompt-variants).
- **Consequences:** Two prompt texts to maintain and to keep in replay fixtures. Users can override per model in the catalog.
- **Date:** 2026-10-13

### ADR-015 — Measured context and window-scaled prefix

- **Context:** Local servers often honor far less context than the model supports and truncate silently, which removes the system prompt first. Fixed prefix sizes that fit at 128k leave no room for the task at 32k.
- **Decision:** `served_context` is measured by the probe and lowered at runtime when usage shows truncation; budgets use the smaller of it and `context_window`. Stable prefix sections take a share of the turn budget (whole prefix at most 25%), and the recent raw window shrinks with the window. Compaction is part of the core loop. Details in [context-engine.md](./context-engine.md#small-windows) and [providers.md](./providers.md#local-and-self-hosted-servers).
- **Consequences:** The prefix differs per model binding (still stable within one). The Ollama adapter must send `num_ctx`.
- **Date:** 2026-10-13

### ADR template (for new entries)

```markdown
## ADR-00X Title
- **Context:** ...
- **Decision:** ...
- **Consequences:** ...
- **Date:** ...
```

## Spec checklist

What the spec still needs before code depends on it:

- [x] CLI surface — commands, flags, TUI wireframes ([cli.md](./cli.md))
- [ ] Sequence diagrams per journey: fix test, small feature, lint cleanup, question
- [ ] `models.yaml` checked against live provider pricing, with capability blocks for each shipped model
- [ ] `compact` and `standard` prompt texts written
- [ ] Probe pass thresholds per step
- [ ] Loop-detector thresholds checked against recorded traces
- [ ] Eval basket run across the model matrix: baseline vs muxon on same SKUs. The runner is `muxon eval basket` (hidden checks, trials, DeepInfra + frontier matrix). This box stays open until that live matrix has been run.
- [ ] ADR-003 finalized
- [x] Go `internal/` package skeleton matching component docs ([implementation-plan.md](./implementation-plan.md#repository-layout-target))

## Research references

- [Harness landscape](../research/02-harness-landscape.md)
- [Awesome CLI survey](../research/03-awesome-cli-coding-agents-survey.md)
- [DeepInfra](../research/01-deepinfra.md)
- [arXiv:2609.20804](https://arxiv.org/abs/2609.20804) — harness component ablations

## Related

- [README.md](./README.md) — overview and diagram
