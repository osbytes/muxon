# Observability

Observability makes the harness **debuggable**, **cost-transparent**, and **regression-testable** without storing prompt bodies on a server.

## Goals

- **Local-first JSONL trace** per session (replay, support, eval).
- **Per-turn metrics**: input/output/cached/reasoning tokens, USD estimate (when rates exist in manifest), model role, provider id, latency.
- **Harness-quality metrics** per model: tool-call parse failures, model calls per tool, edit apply strategy, cache hit rate. These are what tell us whether a capability profile is tuned right.
- **Eval hooks** for benchmark basket (see [research](../research/02-harness-landscape.md#12-suggested-eval-basket-prove-harness)).

## Trace storage

- Path: `.muxon/sessions/<session-id>/trace.jsonl` (gitignored by default).
- Verify logs: `.muxon/sessions/<session-id>/verify-<n>.log`, redacted, referenced by path from `verify.run`.
- **No remote upload** in ZDR mode.
- Optional export: user runs `muxon export-trace` for bug reports (redact secrets).

### Versioning

Every trace starts with `session.start` carrying `schema_version` (semver) and the `muxon` version. Minor versions only add fields; readers ignore unknown fields. Resume and replay refuse a newer **major** version with a message naming the version needed. Checkpoints carry the same field.

## Event types

| Event | Fields |
|-------|--------|
| `session.start` | schema_version, muxon_version, workspace, profile, permission tier, trusted, base_ref, dirty |
| `goal.start` | goal_id, task_type, router_confidence |
| `state.transition` | from, to, reason |
| `model.request` | role, provider_id, api, model_id, tool_protocol, structured_output, prompt_variant, served_context, estimated_input_tokens, prefix_hash (no body in strict ZDR export) |
| `model.response` | usage (input, cached, cache_write, output, reasoning), usage_estimated, truncated_by_server, finish_reason, cost_usd, provider_cost_usd, latency_ms |
| `tool.call` | name, args_hash, phase, parse_ok, constrained, salvaged, salvage_step, repair_attempt |
| `profile.downgrade` | field (`tool_protocol`, `structured_output`, `reasoning_history`, `served_context`), from, to, cause |
| `loop.detected` | pattern, count, response (`nudge`, `escalate`, `fail`) |
| `tool.result` | ok, truncated, duration_ms, apply_strategy (patch), trust, cancelled |
| `approval` | tool, decision, reason (policy, taint, trust) |
| `verify.run` | kind (baseline, full, lint, test), command, exit_code, passed, duration_ms, failure_count, tests_run, log_path |
| `compact.run` | level, tokens_before, tokens_after |
| `escalate` | from_model, to_model, reason, escalation_generation |
| `interrupt` | during (stream, tool), action (pause, cancel) |
| `session.end` | outcome (`done`, `failed`, `paused`, `cancelled`), reason, verified, verify_attempts, escalation_generation, tool_rounds, workspace_changed, protected_files_changed, total_cost_usd, total_tokens |

An interactive session with several goals emits `goal.end` (same fields) per goal; `session.end` repeats the last goal's outcome plus session totals. A headless run has one goal.

`reason` on `session.end` is one of: `verified`, `answered`, `verify_exhausted`, `tool_rounds_exhausted`, `tool_parse_exhausted`, `loop_detected`, `budget_exhausted` (with `budget`: tokens, usd, wall_clock), `approval_required`, `user_cancel`, `workspace_untrusted`, `no_verify_command`.

Strict ZDR profile: omit message bodies from events; keep hashes and sizes only.  
Dev profile: optional full prompt logging locally (required for trace replay fixtures).

## Derived metrics

| Metric | From | Why |
|--------|------|-----|
| `cache_hit_rate` | cached / input tokens | Detects prefix instability; `prefix_hash` changes within a model binding are a bug |
| `model_calls_per_tool` | model.request vs tool.call | Cost of `two_step` on this model |
| `malformed_call_rate` | tool.call.parse_ok | Whether the profile's `tool_protocol` fits the model |
| `salvage_rate` | tool.call.salvaged | High means the protocol or `text_tool_dialect` is wrong, even if calls succeed |
| `loop_rate` | loop.detected per goal | How often the model gets stuck; compare across models and prompt variants |
| `truncation_rate` | model.response.truncated_by_server | Server is honoring less context than the profile claims |
| `percent_well_formed` | patch apply_strategy ∈ {exact, whitespace} | Edit-format fit; fuzzy applies counted separately |
| `cost_per_done` | session.end | Product metric |

## CLI cost report

Commands:

- `muxon cost` — session totals  
- `muxon cost --last` — last task block  

Display example:

```text
Session: 4.2M in (3.1M cached) / 180k out (40k reasoning)  ~$1.14  (worker 92%, escalated 8%)
```

## Eval integration

- `muxon eval run task.json` copies the fixture into a fresh temp workspace, runs the director, then runs the task's hidden `check` (files copied in only after the agent exits). Reward is 1 only when that check exits 0. A timeout is reward 0 with `timeout_hit`.
- `muxon eval basket --matrix evals/matrices/default.yaml --trials 3` writes `.muxon/eval/runs/<run-id>/results.jsonl` and `run.json`. `--budget-usd` stops scheduling once spent cost reaches the cap.
- `muxon eval report <run-id>` groups by model and capability overrides and prints a task-clustered 95% bootstrap interval. `muxon eval compare <run-a> <run-b>` pairs tasks and prints an exact McNemar p-value.
- `muxon run --eval tasks/fix-test-001.json` still appends one row to the workspace `.muxon/eval/results.jsonl` from `session.end`. That path is not the hidden grader.
- Weekly workflow (`.github/workflows/eval.yml`) runs the DeepInfra slice. The frontier row is manual. A harness change that helps the frontier row while hurting the 9B row is a regression.
- How those rows become a score, and how SWE-bench gets graded outside this binary: [benchmarks.md](../benchmarks.md).

Trace schema subset aligned with opencode-harness ideas (compatibility optional).

## Trace replay (harness self-tests)

Dev-profile traces double as **golden fixtures** for deterministic loop tests:

- A `replay` provider serves recorded `model.response` bodies in order. Each request is matched by sequence number and the hash of its assembled messages.
- A hash mismatch fails the test and prints the prompt diff. That catches unintended prompt-layout and cache-prefix changes, not just behavior changes.
- Tests assert the sequence of state transitions, tool calls, and the `session.end` outcome. Tool side effects run against a fixture repo in `testdata/`.
- `muxon trace record` captures a fixture; `muxon trace replay` re-runs one. Fixtures are re-recorded deliberately when the prompt layout changes on purpose.

See [tech-stack.md](./tech-stack.md#testing-lint-ci).

## Influences

- opencode-harness JSONL traces, MartinLoop signed receipts, Agent AFK `trace show`, Claudescope analytics (read-only across agents).

## Related

- [session-director.md](./session-director.md)  
- [model-router.md](./model-router.md)  
- [security-and-policy.md](./security-and-policy.md) — redaction
