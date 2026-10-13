# API approach

Closed-source Go service. The open-source harness calls it with a thee API key. The service calls DeepInfra with our key.

## Role

The API is a **metered inference front door**, not a second harness.

| In the API | Not in the API |
|------------|----------------|
| Auth, quotas, spend caps | Agent loop, tools, shell, repo files |
| Model allow-list | Prompt construction, compaction, skills |
| Upstream call to DeepInfra | Local traces |
| Token and dollar metering | User source code |

The harness already speaks OpenAI-compatible chat. Pointing `base_url` at this API is the integration. No private SDK is required.

## Request path

```text
1. CLI builds messages (local).
2. POST /v1/chat/completions
     Authorization: Bearer <thee_api_key>
     body: model, messages, tools, stream, max_tokens
3. API checks key, balance, model allow-list, rate limit.
4. API forwards to DeepInfra with our key.
     Optional: service_tier, prompt_cache_key (session id only)
     Passed through untouched: response_format, sampling, stop, reasoning params
5. API streams tokens back unchanged, including tool_calls and reasoning_content.
6. API writes a usage row from the provider usage object.
7. Request body and response text are discarded.
```

Streaming is required. Coding-agent turns are long; the CLI must render tokens as they arrive.

## Go service shape

Separate module from the harness. Suggested layout:

```text
cmd/api/                 # process entry
internal/
  httpapi/               # routes, auth middleware
  upstream/              # DeepInfra client (stream copy)
  meter/                 # usage rows, spend caps
  models/                # allow-list + public prices
```

| Concern | Choice |
|---------|--------|
| Language | Go (same major version policy as the harness) |
| HTTP | `net/http` plus a small router ([chi](https://github.com/go-chi/chi)) |
| Upstream | `net/http` streaming copy to `https://api.deepinfra.com/v1/openai` |
| Config | Env: `DEEPINFRA_API_KEY`, database URL, listen addr |
| Logging | `log/slog`. Request id, model, status, latency, tokens. Never bodies. |

One process is enough. No agent runtime inside the API.

### Upstream through a gateway

The `upstream` package can point at DeepInfra directly or at a LiteLLM-style gateway that fans out to several providers. A gateway adds routing and fallback we do not have to write, but it normalizes requests, and normalization is where small-model features get lost. Whatever sits upstream must:

- Pass `response_format` with `json_schema` (and grammar parameters for servers that take them) to the backend, not drop or rewrite them.
- Return `reasoning_content` and `<think>` text as the backend sent them. The harness decides what to strip.
- Report cached and reasoning tokens in usage.
- Keep per-model sampling parameters, including `top_k`, `min_p`, and `repetition_penalty`, which some gateways discard as non-OpenAI fields.

These are checked by a contract test that runs the harness probe against the API on every deploy. If a gateway cannot meet them for a model, that model's published capability block says so (for example `structured_output: none`), so the harness adapts instead of failing.

## Auth

- User creates a **thee API key** in the closed dashboard or with `POST /v1/keys`.
- Harness stores it like any other BYOK secret (`THEE_API_KEY`).
- Keys are hashed at rest. The DeepInfra key exists only in this service's environment.
- Revoke a thee key without rotating the upstream key.

## Models

The API exposes a **curated list**, not all of DeepInfra.

- Public prices can be higher than DeepInfra list (our margin) or equal (subscription covers margin). Pick one before launch; the protocol is the same.
- Unknown model ids return 404.
- Router / worker / escalator ids in the OSS manifest map to these ids when the active provider is `thee`.
- `GET /v1/models` returns each model's capability block (tool protocol, structured output, reasoning history, prompt variant, sampling, served context), produced by running `thee models probe` against our own endpoint. Harness users on our API get a tuned profile without configuring anything; local catalog values still win.

## Metering

Persist only:

- key id, timestamp, model
- input, output, cached token counts
- computed USD
- HTTP status, latency

Use that for balances, invoices, and spend caps. Caps return **429** before the upstream call when the key is exhausted.

Idempotency: a client `X-Request-Id` dedupes a retry so a dropped stream is not double-charged when we can prove the first call did not start.

## Zero data retention

Always. There is no retention mode to turn on.

We do not store prompts, completions, tool arguments, or code. They exist in memory only while the request is in flight, then they are dropped. We do not train on them. Logs carry a request id, model, status, latency, and token counts for billing. They do not carry content.

Upstream calls stay on open models we serve ourselves. We do not add a route that requires a vendor to keep the prompt.

## Deploy

Stateless HTTP, so it can run as a small always-on service or scale-to-zero if cold start is acceptable for a CLI. Streaming and long tool-call turns favor **a small always-on pool** over a platform that kills connections at a few seconds.

DeepInfra stays the upstream until a second provider is worth the code. Add it behind the same `upstream` interface; the public API does not change.

## Failure behavior

| Case | API behavior |
|------|----------------|
| Bad thee key | 401, no upstream call |
| Over spend cap | 429, no upstream call |
| Model not on allow-list | 404 |
| DeepInfra 429 / overload | Pass status through, or one retry on our side |
| Client disconnect | Cancel upstream. Bill only tokens already reported, if any. |

The harness treats this provider like any other OpenAI-compatible host: backoff, then optional failover to a user-configured BYOK provider.

## Related

- [README.md](./README.md)
- [boundary.md](./boundary.md)
- [Harness providers](../harness-architecture/providers.md)
