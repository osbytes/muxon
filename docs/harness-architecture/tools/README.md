# Tools

The model does not edit the repo by talking. It calls tools. How they run, and which permission tier they need, is [../tool-runtime.md](../tool-runtime.md). Each comparison lists thee beside the other harnesses. **Closer to** says `closest` on one of them and is blank everywhere else.

## Tool calling on small models

OpenCode puts a wide tool surface in every prompt and expects several calls in one reply. That holds on a frontier model. On the worker we actually run, it usually breaks: the model picks the wrong tool, emits arguments that miss the schema, or spends the window reading schemas.

The harness still has every tool below. A single turn uses only a few of them.

The session director chooses the active **phase** from the current state. Planning: `search`, `list`, `read`, `repo-map`, `ask_user`. Editing: `read`, `search`, `patch`, `write`, `delete`, `test`, `todo`, `finish`. Verification is not a tool phase: the harness runs the verify command itself and hands the model a `verify_result` ([verifier.md](../verifier.md)); the model goes back to editing. LSP, git, skills, and the rest are real tools that join a phase when the state needs them.

The phase is one line in the volatile tail of the prompt (`Active: read, patch, ...`). It never changes the cached prefix, and the runtime rejects a call outside it. See [context-engine.md](../context-engine.md#tools-on-the-wire).

Each schema is a few flat fields. Closed choices are enums, such as a `git` action or a line range. Deleting a file is `delete`. Committing is `git`.

### Tool calling protocols

How a call is made is a property of the bound model (`tool_protocol` in its [capability profile](../model-router.md#capability-profile)), not of the harness. The same tools and runtime sit behind all three.

| Protocol | For | How a call works |
|----------|-----|------------------|
| `native` | Models with reliable function calling (frontier, strong open coders) | Full tier tool set in `tools[]`, constant for the session. One native call per reply, or several when `parallel_tool_calls` is set. |
| `two_step` | Small or unprofiled models (default when unknown) | 1. The prompt lists tools by name and one line; the model names **one**. 2. The next request carries that tool's flat schema and one example; the model fills the arguments. |
| `fenced` | Models whose native tool calling is broken | Same two steps, but each reply is one fenced block with the same fields. The runtime parses the block. |

`two_step` costs about two model calls per tool. That is the price of reliability on small models, and it is why it is not the default for models that do not need it. Observability tracks `model_calls_per_tool` and malformed-call rate per model so the profile can be tuned (`thee models probe`).

In every protocol the file is written by `patch` or `write`. Assistant text is never saved to disk.

### Constrained decoding

The cheapest malformed call is the one the model cannot produce. When the model's `structured_output` allows it, the harness constrains the reply at decode time instead of hoping it parses:

| `structured_output` | Sent with the request | Typical backends |
|---------------------|-----------------------|------------------|
| `json_schema` | `response_format: {type: json_schema, strict: true}` with the tool's schema | OpenAI, DeepInfra and vLLM guided JSON, Ollama `format`, many gateways |
| `grammar` | A GBNF grammar (or the server's equivalent) for the tool call or fenced block | llama.cpp, vLLM, LM Studio |
| `json_object` | `response_format: {type: json_object}`; valid JSON, schema still checked by the runtime | Older OpenAI-compatible servers |
| `none` | Nothing; the parser and salvage step do the work | Everything else |

Where it is used:

- **`two_step`, step 1:** the reply is constrained to `{"tool": <enum of active-phase tool names>}`. The model cannot name a tool outside the phase.
- **`two_step`, step 2:** the reply is constrained to that tool's argument schema. With a strict schema, step 2 cannot fail to parse, only be wrong.
- **One-step mode:** with `json_schema` or `grammar`, a `two_step` model can be offered a single combined schema (a `oneOf` over the active tools). The probe enables this when it holds the parse rate, which removes the second call per tool.
- **`fenced`:** the grammar covers the fence and the fields.
- **Router classification** and compaction summaries use the same mechanism.

Constraint does not apply to `native` calls; providers with reliable native tools already validate them. If a provider rejects the constraint parameter, the session downgrades `structured_output` one step and retries once, the same way a rejected `tools[]` downgrades `tool_protocol`.

Constrained JSON can still be valid and wrong (a path that does not exist, a line range past the end). The runtime validates semantics and returns a one-line error as it does for any call.

### Salvage before repair

Small models often produce a correct call in the wrong place or with small syntax slips. A repair turn costs a full model call, so the runtime runs a deterministic salvage pass first:

1. **Find the call.** If no native call came back, scan the text for the shapes named in `text_tool_dialect`: Hermes-style `<tool_call>{...}</tool_call>`, Qwen-style XML (`<function=name><parameter=...>`), a JSON object in a fenced block, or a bare top-level JSON object naming a known tool. `auto` tries all of them.
2. **Fix the syntax.** Strip markdown fences and leading prose; remove trailing commas; close a truncated final brace or bracket when the reply hit `max_tokens` only inside the closing tokens; accept single-quoted keys.
3. **Coerce types.** `"12"` to `12` where the schema says integer, a single string to a one-element array, enum values matched case-insensitively.
4. **Validate.** The result must pass the tool schema exactly. Salvage never invents a missing required field or guesses between two tools.

A salvaged call runs normally and is recorded with `salvaged: true` and the step that fixed it ([observability.md](../observability.md#event-types)). A model whose calls are mostly salvaged has the wrong `tool_protocol` or dialect; the metric exists so the profile gets fixed rather than the salvage layer quietly carrying it.

### Repair and escalation

A call that still fails after salvage comes back as the schema error alone. The model gets one repair turn. A second consecutive failure escalates; if the escalated model also fails twice in a row, the goal ends as `failed` ([session-director.md](../session-director.md#caps-and-budgets)).

Results stay shaped and capped: a hit list, a diff stat, an exit code and the last lines. The model asks for more with `read`.

## Comparison

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **thee** | The session state picks a few tools. The protocol follows the model: native calls for capable models, name-then-schema for small ones, a fenced block when native calling is broken. Decoding is constrained to the schema where the backend allows it, in-text calls are salvaged, then one repair, then escalate. |  |
| **OpenCode** | Every tool schema is in the prompt. The model can return several native calls in one reply. The loop is aimed at frontier models. |  |
| **Aider** | The model returns a diff. The harness parses and applies it. A cheaper model can be the editor. |  |
| **Ante** | A profile chooses which tools are exposed, a handful or a full set. The daemon runs the call. Local models are a supported path. | closest |
| **Pi** | Enabled tools are in the prompt as schemas. The model calls them through the provider’s tool API. |  |
| **Goose** | Extensions and MCP supply the tools. The model calls whatever is enabled. |  |

## Workspace

| Tool | What it does |
|------|----------------|
| [search](./search.md) | Find lines without reading whole files |
| [list](./list.md) | List a directory or glob |
| [read](./read.md) | Read a bounded slice of one file |
| [repo-map](./repo-map.md) | A symbol map of the repo |
| [lsp](./lsp.md) | Definition, references, diagnostics |

## Edits

| Tool | What it does |
|------|----------------|
| [patch](./patch.md) | Change an existing file with a unified diff |
| [write](./write.md) | Create a file, or replace a small one |
| [delete](./delete.md) | Delete or move a file |
| [git](./git.md) | Status, diff, commit, worktree |

## Run

| Tool | What it does |
|------|----------------|
| [test](./test.md) | Run a targeted subset of the project's tests |
| [shell](./shell.md) | Run one command (debug, `full` tier) |
| [process](./process.md) | Start a long-running process and read it later |
| [browser](./browser.md) | Fetch a URL or drive a page |

## Control

| Tool | What it does |
|------|----------------|
| [todo](./todo.md) | Keep a task list for the session |
| [ask_user](./ask_user.md) | Ask the user one question and wait |
| [skills](./skills.md) | Load one procedure by name |
| [memory](./memory.md) | Read and write notes that outlive the turn |
| [subagent](./subagent.md) | Spawn a read-only or writing child agent and take back its result |
| [mcp](./mcp.md) | Call a tool from an attached MCP server |
| [hooks](./hooks.md) | Run project code before or after a tool |
| [checkpoint](./checkpoint.md) | Save or restore the workspace |
| [finish](./finish.md) | End the task; requires a verifier pass if files changed |

Streamed reply text is never written to disk. File changes go through `patch`, `write`, `delete`, and `git`.
