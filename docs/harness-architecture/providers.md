# Providers

Providers are **adapters** from the harness to inference backends. The default path is **OpenAI-compatible chat completions** (tools + streaming), which covers open-model hosts, local servers, gateways, and our API. Anthropic and OpenAI get native adapters as well, because their compatibility layers drop the features (explicit caching, signed thinking, reasoning state) that make them worth paying for. See [Adapters](#adapters).

## Bring your own keys (BYOK)

**muxon never requires our API keys or a muxon account** to run inference. Users authenticate to whatever backends they choose; the harness only reads credentials from the user’s machine (or their team secret store).

The harness lets users bring their own key. Our own inference is the provider we want them to select, specified in [../api/README.md](../api/README.md): lower price, and prompts, completions, and code are never retained. The client protocol is the same OpenAI-compatible call either way.

| Principle | Detail |
|-----------|--------|
| **User-owned credentials** | Keys live in env vars, OS keychain, or a user-level config file—not embedded in the binary. |
| **Any supported host** | If it speaks OpenAI-compatible chat completions (+ tools), you can register it. |
| **Default ≠ exclusive** | Shipped `models.yaml` recommends frugal OSS routes (often DeepInfra); users replace providers, models, and rates freely. |
| **No key escrow** | CLI talks directly to the provider; we do not proxy or store keys on our servers. Choosing our API means using a muxon key against our endpoint, not handing us a vendor key. |
| **Multi-provider** | Different roles (router / worker / escalated) may use different providers and keys in one session. |

### Credential resolution order

1. Per-provider `api_key_env` in `models.yaml` or `catalog.json` (e.g. `DEEPINFRA_API_KEY`). At startup `muxon` loads the nearest `.env` at or above the current directory (stopping at the directory that holds `.git`). It never overrides variables already set in the environment.
2. Generic fallbacks where documented (e.g. `OPENAI_API_KEY` for OpenAI-compatible base URLs)—explicit in config, not magic.
3. OS keychain entries via `muxon auth login <provider>`.
4. **(team)** CI/org secret injection—same env-based contract.

`muxon doctor` should report which providers are configured (key present, base URL reachable) without printing secret values.

### User provider registry: `catalog.json`

Ante-style file (default `~/.muxon/catalog.json` or path in `muxon.toml`) for providers and models **without** waiting for a release:

```json
{
  "providers": {
    "my-deepinfra": {
      "base_url": "https://api.deepinfra.com/v1/openai",
      "api_key_env": "DEEPINFRA_API_KEY"
    },
    "openrouter": {
      "base_url": "https://openrouter.ai/api/v1",
      "api_key_env": "OPENROUTER_API_KEY"
    },
    "company-gateway": {
      "base_url": "https://llm.internal.example/v1",
      "api_key_env": "CORP_LLM_KEY"
    }
  }
}
```

Model entries in `models.yaml` reference `provider: my-deepinfra` (or inline overrides).

Project `muxon.toml` cannot register providers or redirect base URLs; that would let a repo route your code to its own endpoint. Provider config is user-level only ([security-and-policy.md](./security-and-policy.md#workspace-trust)).

### Our API as a provider

The [muxon API](../api/README.md) (pay-as-you-go credits) is one provider entry like any other: `base_url` plus `MUXON_API_KEY`. It is **optional**. BYOK remains fully supported for solo devs, air-gapped, and teams with existing vendor contracts, and choosing our API never disables direct BYOK.

## Recommended defaults (not lock-in)

| Provider | Typical use | Notes |
|----------|-------------|--------|
| **DeepInfra** | Default OSS worker/router SKUs | User’s `DEEPINFRA_API_KEY`; caching, flex tier — [research](../research/01-deepinfra.md) |
| **OpenRouter** | Failover / model variety | User’s `OPENROUTER_API_KEY` |
| **Anthropic** | Frontier escalation | User’s `ANTHROPIC_API_KEY`; native `anthropic_messages` adapter |
| **OpenAI** | Frontier escalation | User’s `OPENAI_API_KEY`; native `openai_responses` adapter for reasoning models, chat completions otherwise |
| **OpenCode Zen and similar** | Curated hosted models | OpenAI-compatible; user’s key |
| **Ollama / vLLM / llama.cpp / LM Studio** | Offline / on-prem | Often no key; `base_url` only. See [Local and self-hosted servers](#local-and-self-hosted-servers) |
| **LiteLLM-style gateways** | Team or company proxy | OpenAI-compatible; may publish capabilities on `GET /v1/models` |
| **catalog.json** | Custom gateways | Internal proxies, Azure OpenAI, Bedrock OpenAI shim, etc. |

## Adapters

Every provider entry names its wire API with `api:` (default `openai_chat`). The rest of the harness only sees the normalized request and response below; adapters translate.

| `api` | For | What the adapter handles |
|-------|-----|--------------------------|
| `openai_chat` | Open-model hosts, local servers, gateways, our API | `tools[]`, `response_format`, `stream_options.include_usage`, `prompt_cache_key`, `reasoning_content` on the delta |
| `anthropic_messages` | Anthropic, and gateways that expose the Messages API | `system` as a separate field; `cache_control` breakpoints at the context engine's cache boundary; `tool_use` / `tool_result` blocks; `thinking` blocks with signatures kept across tool turns; `cache_creation_input_tokens` / `cache_read_input_tokens` usage |
| `openai_responses` | OpenAI reasoning models | Reasoning items carried between turns client-side (`store: false` with encrypted reasoning content, so nothing is retained at the vendor and no `previous_response_id` is needed); `reasoning.effort`; function call items; usage with `reasoning_tokens` |

The normalized interface:

```text
Provider.complete(CompletionRequest) -> CompletionResponse
Provider.stream(CompletionRequest) -> Stream<Chunk>

CompletionRequest {
  model_id,
  messages[],          // normalized: role, content parts, tool calls, tool results, reasoning parts
  tools?,              // native protocol only
  constraint?,         // json_schema | grammar | json_object, from structured_output
  max_output_tokens,
  sampling,            // from the capability profile; overridable per role
  reasoning?,          // effort or budget, when the profile supports it
  cache?,              // prompt_cache_key, or the index of the cache boundary for explicit markers
  service_tier?,       // standard | flex | priority
  fail_fast?,
}

CompletionResponse {
  text,
  tool_calls[],
  reasoning?,          // text and, where present, the opaque signature or encrypted item
  finish_reason,       // stop | tool_calls | length | content_filter
  usage,               // normalized; see Usage parsing
}
```

A model reached through a gateway keeps the capability profile of the model, not the gateway. If a gateway translates Anthropic into OpenAI-compatible chat, the entry uses `openai_chat` and the profile records what survives the translation (usually: no explicit caching, `reasoning_history: strip`).

## Reasoning content

Reasoning models return their thinking in different places, and each provider has its own rule for what must be sent back. Getting this wrong breaks multi-turn tool use silently: the reply looks fine, then the next turn errors or the model loses track.

| Where it shows up | Examples | `reasoning_history` default |
|-------------------|----------|-----------------------------|
| `<think>...</think>` inside the text | Qwen3, DeepSeek-R1 distills, many local models | `strip` |
| `reasoning_content` field on the message or delta | DeepSeek API, vLLM and DeepInfra reasoning parsers | `strip` |
| `thinking` blocks with a signature | Anthropic extended thinking | `keep_signed` |
| Reasoning items, possibly encrypted | OpenAI Responses | `keep_signed` |

Rules:

- **Always separate it.** Reasoning never reaches the tool parser, the salvage parser, or the user-facing answer. `<think>` blocks are cut out of the text before anything else reads it; an unclosed `<think>` at `max_tokens` means the turn produced no answer and is retried with a higher output cap once.
- **`strip`:** prior reasoning is dropped from history. Required by DeepSeek-style APIs (they reject it) and recommended by Qwen for multi-turn. Saves context on small windows.
- **`keep`:** prior reasoning stays as plain text in history. Rare; only when the probe shows the model degrades without it.
- **`keep_signed`:** the opaque block or item is sent back unchanged with the tool result that followed it, as the provider requires. The engine never edits or summarizes these; compaction drops them whole once their tool round leaves the recent window.
- Reasoning tokens are metered either way ([cost accounting](./model-router.md#cost-accounting)). In strict ZDR export, reasoning text is omitted like any other body.

## Local and self-hosted servers

Most "small models don't work in this harness" reports come from the server, not the model. The harness checks the usual failures instead of trusting defaults:

- **Silent truncation.** Ollama's default context is far smaller than the model's window and it drops the start of the prompt without an error, which deletes the system prompt and tool catalog first. The adapter sends `num_ctx` (Ollama `options`) sized to the turn budget, and the probe measures `served_context` for every local endpoint. When usage shows `prompt_tokens` well below the harness estimate, the turn is flagged as truncated and the session drops to the measured window.
- **Chat template and tool parser.** vLLM and llama.cpp only return native tool calls when the server runs with the right tool parser and template. Without them, calls arrive as text. The salvage parser handles this, and `muxon doctor` reports when a native-profile model is producing text calls.
- **Stop sequences.** Some templates leak end-of-turn tokens or continue past them. Per-model `sampling.stop` covers this.
- **Concurrency.** Local servers often serve one request at a time. The harness does not send concurrent model calls (subagents, background summarization) to an endpoint marked `max_concurrency: 1`; they queue.
- **No usage.** Some servers omit usage in streams. Token budgets then use the harness estimate, marked as estimated in the trace.

## DeepInfra-specific behavior

| Feature | Harness use |
|---------|-------------|
| **Prompt caching** | Stable prefix from [context-engine.md](./context-engine.md); read `cached_tokens` from usage |
| **Flex tier** | Background summarization, eval jobs |
| **Priority** | Optional paid team SLA |
| **fail_fast** | Failover to secondary provider when overloaded |
| **Tool calling** | Native path for models whose profile says `tool_protocol: native`; otherwise two-step or fenced over plain completions |

## ZDR mode

When `zdr_strict = true` in config:

- Block manifest models that proxy to Google/Anthropic on DeepInfra.
- Prefer OSS weights only.
- Optional: disallow cloud providers entirely (local only).

Harness still does not log bodies; provider choice reduces subprocessors.

## Secrets

- **BYOK only on the client path:** resolve keys at runtime; never commit to repo or session traces.
- Prefer `api_key_env` indirection in YAML/JSON over literal keys in config files.
- Keychain storage for interactive dev; env vars for CI and headless.
- Never write keys into traces, logs, or error reports (redact on export).

## Streaming

- Session director consumes stream for UX (token display).
- Tool calls: accumulate until finish_reason complete; validate JSON args before tool runtime. Text is buffered for the salvage parser when no native call arrived.
- Reasoning deltas (`reasoning_content`, `<think>` spans, thinking blocks) are routed to a separate channel as they stream, so the UI can show or hide them and parsers never see them.
- `finish_reason: length` inside a tool call means the arguments were cut off. Salvage may close a final brace; otherwise the call is retried once with a larger `max_tokens`, not sent to repair.
- Cancellation: closing the request context aborts the HTTP stream; partial tool arguments are discarded ([session-director.md](./session-director.md#interrupts-and-steering)).

## Usage parsing

Normalize every provider's usage into one record:

| Field | `openai_chat` | `anthropic_messages` | `openai_responses` |
|-------|---------------|----------------------|--------------------|
| `input_tokens` | `prompt_tokens` (includes cached) | `input_tokens + cache_read_input_tokens + cache_creation_input_tokens` | `input_tokens` |
| `cached_tokens` | `prompt_tokens_details.cached_tokens` (or provider equivalent) | `cache_read_input_tokens` | `input_tokens_details.cached_tokens` |
| `cache_write_tokens` | — | `cache_creation_input_tokens` (priced at the cache-write rate) | — |
| `output_tokens` | `completion_tokens` (includes reasoning) | `output_tokens` (includes thinking) | `output_tokens` |
| `reasoning_tokens` | `completion_tokens_details.reasoning_tokens` when present | not reported separately | `output_tokens_details.reasoning_tokens` |
| `provider_cost_usd` | `usage.cost` (OpenRouter), gateway cost headers, or our API | — | — |

Request `stream_options.include_usage` when streaming on `openai_chat`. Cost formula: [model-router.md](./model-router.md#cost-accounting); a model with `cache_write_per_m` prices cache writes at that rate.

## Error handling

| Error | Action |
|-------|--------|
| 429 `engine_overloaded` + fail_fast | Retry on backup provider |
| 429 rate limit | Backoff + user message |
| Context length 400 | Trigger context engine compaction, retry once; lower `served_context` for the session to the size that failed |
| Tool parse error | Salvage, then return error to model; do not crash session (rules in [tools/README.md](./tools/README.md#salvage-before-repair)) |
| Provider rejects `tools` | Mark the model `tool_protocol: two_step` for the session and retry once |
| Provider rejects `response_format` or grammar | Downgrade `structured_output` one step for the session and retry once |
| Provider rejects reasoning in history | Switch `reasoning_history` to `strip` for the session and retry once |
| Reply truncated by the server (usage far below estimate) | Treat as `served_context` too small; compact to the measured size and retry once |

## Manifest: `models.yaml`

Shipped as a **starter** manifest; users override path via `muxon.toml` or `--models-manifest`. All `provider` names must exist in `providers:` or `catalog.json`.

```yaml
version: 1
providers:
  deepinfra:
    base_url: https://api.deepinfra.com/v1/openai
    api_key_env: DEEPINFRA_API_KEY   # user sets in their environment
  openrouter:
    base_url: https://openrouter.ai/api/v1
    api_key_env: OPENROUTER_API_KEY
  anthropic:
    api: anthropic_messages
    base_url: https://api.anthropic.com
    api_key_env: ANTHROPIC_API_KEY
  ollama:
    base_url: http://localhost:11434/v1
    max_concurrency: 1
models:
  - id: worker-primary
    provider: deepinfra
    model: Qwen/Qwen3-Coder-480B-A35B-Instruct-Turbo
    input_per_m: 0.30      # for cost display; user may edit if rates change
    output_per_m: 1.00
    cached_input_per_m: 0.10
    replaced_by: null
    capabilities:
      context_window: 262144
      max_output_tokens: 16384
      tool_protocol: native      # native | two_step | fenced
      structured_output: json_schema  # json_schema | grammar | json_object | none
      text_tool_dialect: qwen_xml     # none | hermes | qwen_xml | json_block | auto
      parallel_tool_calls: false
      edit_format: udiff         # udiff | search_replace
      vision: false
      reasoning: none            # none | tokens | effort
      reasoning_history: strip   # strip | keep | keep_signed
      prompt_variant: standard   # compact | standard
      sampling: { temperature: 0.7, top_p: 0.8, top_k: 20, repetition_penalty: 1.05 }
      tokenizer: hf:Qwen/Qwen3-Coder-480B-A35B-Instruct
      prompt_cache: key          # auto | key | explicit | none
  - id: router-small
    provider: deepinfra
    model: meta-llama/Meta-Llama-3.1-8B-Instruct-Turbo
    input_per_m: 0.02
    output_per_m: 0.03
    capabilities:
      context_window: 131072
      tool_protocol: fenced
      structured_output: json_schema
      edit_format: search_replace
      prompt_variant: compact
  - id: worker-local
    provider: ollama
    model: qwen3-coder:30b
    capabilities:
      context_window: 262144
      served_context: 32768      # measured by `muxon models probe`; adapter sends num_ctx
      tool_protocol: two_step
      structured_output: json_schema  # Ollama `format`
      text_tool_dialect: qwen_xml
      edit_format: search_replace
      prompt_variant: compact
      sampling: { temperature: 0.7, top_p: 0.8, top_k: 20, repetition_penalty: 1.05 }
  - id: escalated-frontier
    provider: anthropic
    model: claude-sonnet-4-5
    input_per_m: 3.00
    output_per_m: 15.00
    cached_input_per_m: 0.30
    cache_write_per_m: 3.75
    capabilities:
      context_window: 200000
      tool_protocol: native
      parallel_tool_calls: true
      edit_format: udiff
      reasoning: tokens
      reasoning_history: keep_signed
      prompt_cache: explicit
  - id: worker-fallback
    provider: openrouter
    model: qwen/qwen3-coder
    input_per_m: null      # observability: estimate unavailable unless rates set
    output_per_m: null
    # no capabilities block: conservative defaults (two_step, compact prompt, search_replace, 32k)
```

If `input_per_m` / `output_per_m` are omitted, **observability** still records token counts but may show `$—` until the user adds rates. A model with reasoning billed at a different rate sets `reasoning_per_m`.

### Capability fields

What each field drives is in [model-router.md](./model-router.md#capability-profile). Rules:

- Missing fields take conservative defaults, so an unknown model works, just less efficiently. `parallel_tool_calls` stays off.
- Native models connected through `/providers` are the exception: they are created with `tool_protocol: native` and `parallel_tool_calls: true`. A later `/model` switch on that provider keeps the flag. One reply may then contain up to 8 calls, run in order.
- `catalog.json` entries can carry the same `capabilities` block; user values override the shipped manifest.
- `muxon models probe <id>` measures served context, structured-output support, tool protocols, edit formats, and reasoning behavior against the real endpoint, and writes a suggested block to the catalog ([Probe](./model-router.md#probe)).
- A gateway that publishes capabilities on `GET /v1/models` (our API, or a LiteLLM-style proxy that exposes model info) fills defaults automatically; local config still wins.

## Related

- [model-router.md](./model-router.md)  
- [observability.md](./observability.md) — usage accounting  
- [security-and-policy.md](./security-and-policy.md)
