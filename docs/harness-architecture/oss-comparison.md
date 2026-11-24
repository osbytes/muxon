# Open-source harness comparison

Each table lists muxon beside five open-source references. **Closer to** is blank except on the one harness the design is closest to. That cell says `closest`.

| Harness | Why it's here |
|---------|----------------|
| [OpenCode](https://github.com/anomalyco/opencode) | Large multi-surface agent |
| [Aider](https://github.com/Aider-AI/aider) | Diff-first editor |
| [Ante](https://github.com/AntigmaLabs/ante-preview) | Model-agnostic daemon |
| [Pi](https://github.com/badlogic/pi-mono) | Minimal toolkit |
| [Goose](https://github.com/aaif-goose/goose) | Extension-driven local agent |

## Small-model handling

The question this product exists to answer: what does each harness do when the model is a 27B or 8B open model instead of a frontier one?

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **muxon** | A capability profile per model drives tool protocol, constrained decoding, salvage of in-text calls, edit format, prompt variant, sampling, reasoning handling, and measured context. A probe fills it in. Loops are detected and cut off. |  |
| **OpenCode** | One loop tuned for frontier tool calling: every schema in every prompt, several native calls per reply. Small models pick wrong tools and miss schemas. |  |
| **Aider** | Per-model settings choose the edit format and some prompt behavior; the model edits by text, so weak function calling matters less. No tool loop to break. | closest |
| **Ante** | Profiles choose a small or full tool set; local models through llama.cpp are a supported path. |  |
| **Pi** | Provider tool API for every model. No small-model mode. |  |
| **Goose** | An experimental toolshim lets a local model interpret tool calls for a model without native calling. Otherwise extensions assume native tools. |  |

What we take: Aider's per-model edit formats and lenient matching, Ante's profiles and local-model path, Goose's idea that tool calls can arrive as text and be interpreted. What we add that none of them do together: constrained decoding, deterministic salvage, loop detection, measured served context, and an eval that always includes the smallest models.

## Session director

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **muxon** | One state machine: plan, act, tools, verify, escalate, then done, failed, or cancelled under hard caps and a loop detector. Monolith CLI. |  |
| **OpenCode** | Client–server, many sessions, plan/build roles. Done is not a verify state. |  |
| **Aider** | Chat plus git. Architect and editor modes. No escalate or blocked state. |  |
| **Ante** | Daemon owns each turn: prompt, tools, confirm, execute. | closest |
| **Pi** | Small loop plus extensions. Phases stay informal. |  |
| **Goose** | Recipe loop. Pauses for tool permission, not for verify. |  |

## Context engine

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **muxon** | Cache-stable prompt, retrieve instead of pasting the repo, token budget from the measured window, prefix scaled to the window, compact or standard prompt per model, elide then summarize. |  |
| **OpenCode** | LSP and a large window. Compaction assumes a strong model. |  |
| **Aider** | Repo map and files you add. Little formal compaction or cache layout. | closest |
| **Ante** | Profile sets the context. No published token budget. |  |
| **Pi** | Skills and extensions inject context. No budgeted engine. |  |
| **Goose** | Recipes and extensions supply context. Cache layout is not the point. |  |

## Model router

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **muxon** | Cheap router, then a worker, then a bigger model only after failed checks. Tool protocol and edit format follow each model's capability profile. |  |
| **OpenCode** | User picks the model. Zen points at frontier models. |  |
| **Aider** | Strong architect model, cheaper editor. The user picks. Tests do not escalate. | closest |
| **Ante** | Any model, including local. A profile chooses. No evidence router. |  |
| **Pi** | One model per run. |  |
| **Goose** | Provider and model are config. No escalator. |  |

## Tool runtime

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **muxon** | The full tool set in [tools/README.md](./tools/README.md): workspace, edits, run, and control. Caps on output. Built-in tools stay when MCP is attached. |  |
| **OpenCode** | Wide surface, including LSP, sessions, and plugins. | closest |
| **Aider** | Diffs and git. The user mostly chooses files. |  |
| **Ante** | A profile can expose a handful of tools or a full set. |  |
| **Pi** | Tools, skills, and MCP. The surface grows by extension. |  |
| **Goose** | Extensions and MCP are how the tool list grows. |  |

## Verifier

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **muxon** | The harness runs the tests itself. Done waits until they pass. Baseline run, test-edit policy, repeated failure escalates, then stops as failed. |  |
| **OpenCode** | The model can run tests. Done is not a verifier state. |  |
| **Aider** | `/test` and lint exist. The user runs them. They do not gate the loop. |  |
| **Ante** | Tests are a tool. Finish does not require them. |  |
| **Pi** | No verifier. |  |
| **Goose** | Shell can run tests. Passing is not required. |  |

## Providers

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **muxon** | BYOK, a priced role manifest, and `catalog.json`. OpenAI-compatible by default, with native Anthropic Messages and OpenAI Responses adapters. Per-model reasoning handling. Local servers checked for truncation. Our API is optional, cheaper, and keeps zero data retention. No embedded local engine. |  |
| **OpenCode** | 75+ providers, Copilot and ChatGPT login, Zen tier. |  |
| **Aider** | Broad BYOK, including local models. Flags and YAML. No priced roles. |  |
| **Ante** | Presets, `catalog.json`, and built-in GGUF. | closest |
| **Pi** | Unified provider API and user keys. No priced manifest. |  |
| **Goose** | Many providers and local models through extension config. |  |

## Observability

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **muxon** | Local JSONL, tokens and dollars per turn, harness-quality rates (parse, salvage, loop, truncation) per model, and an eval across a model matrix. Strict mode omits prompt bodies. |  |
| **OpenCode** | Sessions and share links. Cost is not the product. |  |
| **Aider** | Chat history and some token display. No cost ledger. |  |
| **Ante** | Daemon protocol. Public story is benchmarks. |  |
| **Pi** | Ordinary session logs. |  |
| **Goose** | Session logs. No dollars per task. |  |

## Security and policy

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **muxon** | Read, edit, and full tiers. Workspace trust for repo config and hooks. Secret deny list, argv-only exec, scrubbed env, optional sandbox. Our provider never retains content. |  |
| **OpenCode** | Local execution and a privacy claim. |  |
| **Aider** | Local process and git. You are the policy. |  |
| **Ante** | Daemon confirmations. Local by default. No retention rule for a hosted route. |  |
| **Pi** | Trust is which extensions you load. |  |
| **Goose** | Prompts before a tool runs. Local-first. | closest |

## Related

- [README.md](./README.md)
- [tools/README.md](./tools/README.md)
- [../research/02-harness-landscape.md](../research/02-harness-landscape.md)