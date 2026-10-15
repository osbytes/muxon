# CLI surface

**Status:** Pre-implementation. Defines commands, flags, and interactive layout. The loop itself lives in [session-director.md](./session-director.md); this doc is how users and CI invoke it.

## Goals

| Goal | CLI implication |
|------|-----------------|
| **CLI-first** | One binary; default path is interactive in a repo. |
| **Same loop everywhere** | `thee`, `thee run`, and `thee eval` all drive the same session director. |
| **Cost visible** | Status line and `thee cost` show tokens and $ every session. |
| **CI-friendly** | `thee run` has stable exit codes, `--json`, and no prompts when flags say so. |
| **ZDR** | No upload; traces stay under `.thee/sessions/`. |

Non-goals for v1 CLI: desktop app, plugin marketplace, multi-session dashboard (orchestrators come later).

---

## Entry points

| Invocation | When | Behavior |
|------------|------|----------|
| **`thee`** | User in a repo (default) | Start or resume **interactive** session (TUI). Equivalent to `thee chat`. |
| **`thee chat`** | Explicit REPL | Same as `thee`; optional `[path]` to set workspace root. |
| **`thee run -p "…"`** | Scripts, CI, eval | **Headless** one goal; stdout is progress + final summary; trace on disk. |
| **`thee <subcommand>`** | Setup, ops, debug | No agent loop (see command tree below). |

Global flags (all subcommands that touch a workspace):

| Flag | Effect |
|------|--------|
| `-C`, `--directory` | Workspace root (default: cwd). |
| `--json` | Machine-readable events on stdout (headless and some subcommands). |
| `-q`, `--quiet` | Errors and final outcome only. |
| `-v`, `--verbose` | Extra harness logging (`slog` to stderr). |

Environment mirrors flags where noted: `THEE_TRUST`, `THEE_*` per [security-and-policy.md](./security-and-policy.md#config-precedence).

---

## Command tree (v1)

```text
thee                          # → chat (interactive)
thee chat [path]

thee run -p PROMPT [flags]    # headless goal
thee resume [session-id]      # interactive; latest in cwd if id omitted

thee doctor                   # keys, providers, rg, optional LSP, tool-parser hints
thee cost [--last]            # session totals from trace

thee auth login [provider]    # keychain / catalog; optional thee API key
thee models list|show|probe   # manifest + catalog + probe → suggested profile

thee trust list|add|rm        # ~/.thee/trust.json
thee memory list|show|rm|review

thee trace export|record|replay
thee eval run|basket|report|compare|export-swebench
thee gc                       # prune old shadow refs / sessions

thee serve [flags]            # v1 stub or minimal JSONL; full protocol later

thee version
thee completion bash|zsh|fish
```

**Naming:** Research drafts used `cost-report`; the product command is **`thee cost`** ([observability.md](./observability.md#cli-cost-report)).

### `thee run` flags (headless)

| Flag | Effect |
|------|--------|
| `-p`, `--prompt` | Goal text (required). |
| `--trust` | Honor project verify command, hooks, skills ([security](./security-and-policy.md#workspace-trust)). |
| `--verify-cmd` | Override detected verify command (argv vector; no shell). |
| `--permission` | `read-only` \| `edit` \| `full` (capped by user `max_permission`). |
| `--yolo` | Auto-approve within tier up to `max_permission`; enables `shell` in `full`. |
| `--model` | Force worker SKU (still uses router for classify unless `--no-router`). |
| `--no-plan` | Skip planning phase. |
| `--eval PATH` | Task file; append row to `.thee/eval/results.jsonl`. |
| `--max-tool-rounds`, `--cost-budget-usd`, … | Override `[session]` caps for this run. |

Exit codes: [session-director.md](./session-director.md#exit-codes).

Interactive session inherits the same caps from `thee.toml` and `~/.thee/config.toml`; headless denies anything that would enter `blocked` except where the table allows `ask_user` to continue.

---

## Interactive layout (wireframe)

Bubble Tea TUI ([tech-stack.md](./tech-stack.md#cli--terminal-ui)). Not a pixel mockup — regions and what each shows.

### Default session (transcript mode)

```text
┌─ thee ─ workspace: ~/src/myapp ─ worker: qwen3-coder@deepinfra ─ edit ─ trusted ─┐
│ fix the failing TestParse test                                                    │
├───────────────────────────────────────────────────────────────────────────────────┤
│                                                                                   │
│  [assistant] I'll search for the test and read the parser.                        │
│                                                                                   │
│  ▸ search  pattern="TestParse"  → 3 hits (truncated)                              │
│  ▸ read    pkg/parse_test.go:40-90                                              │
│  ▸ patch   pkg/parse.go  applied (exact)                                          │
│                                                                                   │
│  [harness] verify running: go test ./...                                        │
│  [harness] verify failed (exit 1) — 1 failure, log: .thee/sessions/…/verify-2.log │
│                                                                                   │
│  [assistant] …                                                                    │
│                                                                                   │
├───────────────────────────────────────────────────────────────────────────────────┤
│ goal: change │ state: acting │ rounds 12/40 │ verify 2/6 │ $0.08 │ 420k in (310k  │
│ cached) / 18k out │ Ctrl+C pause · /help                                        │
├───────────────────────────────────────────────────────────────────────────────────┤
│ > _                                                                               │
└───────────────────────────────────────────────────────────────────────────────────┘
```

**Regions**

| Region | Content |
|--------|---------|
| **Header** | Binary version (on first open), workspace path, bound worker model, permission tier, trust. |
| **Transcript** | User messages, assistant text (streamed), tool lines (compact one-liners), harness lines (verify, trust, loop nudge, budget warn at 80%). |
| **Status** | Director state, caps, cost, cache hint; keyboard hints. |
| **Input** | Multiline; Enter submits; Shift+Enter newline (or meta configurable). |

Tool lines use a fixed icon (`▸`) and never dump full file bodies — details live in trace + optional `thee trace export`.

### Approval modal (`blocked`)

```text
┌─ Approval required ───────────────────────────────────────────────────────────────┐
│ Tool: shell (full tier)                                                           │
│ argv: ["npm", "run", "build"]                                                     │
│ Reason: not allowed in edit tier; user requested build                            │
│                                                                                   │
│   [y] allow once   [a] allow for session   [n] deny   [q] cancel goal             │
└───────────────────────────────────────────────────────────────────────────────────┘
```

Trust-first-run uses the same modal pattern with a checklist (hooks, verify command, policy deltas).

### Planning (`planning` / `blocked`)

Plan renders in transcript; full body in `plans/<session-id>.md`. Approve modal: **accept plan** / **edit in $EDITOR** / **skip to acting**.

---

## Slash commands (interactive)

Parsed only in the input line (not passed to the model). Unknown `/foo` is sent as a normal user message.

| Command | Effect |
|---------|--------|
| `/help` | Short command list. |
| `/model [id]` | Switch the worker model. With no argument, pick from the connected provider's model list. |
| `/abort` | Cancel goal (`cancelled`). |
| `/exit` | Quit thee. `/quit` is the same command. |
| `/undo` | Revert last applied patch batch ([session-director](./session-director.md#checkpoints--resume)). |
| `/escalate` | Force worker → escalated if budget allows ([model-router](./model-router.md#escalation-triggers)). |
| `/continue` | After `paused` or `failed`, resume the same history with a fresh tool-round budget. |
| `/cost` | Print session totals (same as status bar, more detail). |
| `/compact` | Request context compaction now (director may still compact automatically). |
| `/trust` | Open trust UI for this workspace. |
| `/permission read-only\|edit\|full` | Lower or raise tier within user cap (raising may prompt). |

Future: `/resume`, `/checkpoint` — defer until loop is stable.

---

## What the user sees per director state

Maps [state machine](./session-director.md#state-machine) to UI:

| State | UI |
|-------|-----|
| `idle` | Empty transcript or “previous goal ended”; input focused. |
| `planning` | “Planning…” spinner; then plan markdown preview. |
| `acting` | Streaming assistant text; input disabled until turn ends. |
| `tooling` | Tool lines append; long tools show elapsed time. |
| `verifying` | Harness line “verify running…”; input disabled. |
| `escalating` | Banner “Escalating to &lt;model&gt; (reason)”. |
| `blocked` | Modal or inline prompt (approval, `ask_user`, interrupt). |
| `done` / `failed` / `cancelled` | Banner with outcome + reason; input accepts **new goal** (same session id, counters reset). |

Interrupts: [session-director.md](./session-director.md#interrupts-and-steering).

---

## Output modes

| Mode | Use |
|------|-----|
| **Human (default)** | Lipgloss-styled transcript; stderr for warnings. |
| **`--json`** | NDJSON on stdout: subset of trace events (`state.transition`, `tool.call`, `tool.result`, `verify.run`, `session.end`) for CI parsers. |
| **Quiet** | Final line: `done` / `failed: verify_exhausted` + exit code. |

`thee run --json` does not include prompt bodies in strict ZDR profile (same as trace export).

---

## First-run and `doctor`

Suggested flow:

1. `thee doctor` — missing `DEEPINFRA_API_KEY` (or catalog provider), `rg` not on PATH, workspace not git (warning only).
2. `thee auth login deepinfra` (or edit `~/.thee/catalog.json`).
3. `cd myrepo && thee` — trust prompt if `.thee/hooks` or verify command present.
4. Type goal or `thee run -p "…"` in CI with `--trust`.

`doctor` never prints secret values; it prints provider id, base URL reachability, and manifest role resolution.

---

## v1 vs later

| Ship in v1 | Defer |
|------------|--------|
| `chat`, `run`, `resume`, `doctor`, `cost`, `auth`, `models list/show`, `trust`, `trace export`, `eval run`, `version`, `completion` | `serve` full IDE protocol |
| TUI transcript + approvals + status bar | Split panes, mouse, themes beyond dark/light |
| Slash: `/help`, `/abort`, `/undo`, `/escalate`, `/cost` | `/checkpoint` UI |
| `models probe` (may be slow; progress UI) | `gc` automation in CI |
| `memory review` if memory tool ships | Rich `memory` TUI |

---

## Related

- [session-director.md](./session-director.md) — states, headless, interrupts  
- [observability.md](./observability.md) — trace path, cost, eval  
- [security-and-policy.md](./security-and-policy.md) — trust, tiers, `--yolo`  
- [tech-stack.md](./tech-stack.md) — cobra, bubbletea  
- [README.md](./README.md) — harness overview  
