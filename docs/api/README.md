# muxon API

Closed-source **Go** service that sits between the open-source harness and inference (default upstream: [DeepInfra](../research/01-deepinfra.md)).

The harness stays public. This API does not.

## Split

| Piece | License | Owns |
|-------|---------|------|
| **Harness** (`muxon` CLI) | Open source | Loop, tools, context, routing policy, local traces |
| **API** (this service) | Closed source | Accounts, our upstream keys, metering, abuse limits |
| **DeepInfra** | Vendor | Model execution |

```text
muxon CLI (OSS)
    │  OpenAI-compatible chat + stream
    ├─ BYOK ──────────────► user's DeepInfra / OpenRouter / local
    └─ muxon API key ──────► api.muxon… (closed, Go)
                                │  our DeepInfra key
                                └─► DeepInfra
```

The open-source harness lets users bring their own key. This API is the inference we want them to use: lower prices than calling the upstream themselves, and zero data retention with no option to store prompts, completions, or code. See [approach.md](./approach.md).

## What the API is

A thin **OpenAI-compatible** proxy:

- `POST /v1/chat/completions` (stream and non-stream)
- `GET /v1/models` (the curated manifest we allow)
- Auth with a **muxon API key**, not a DeepInfra key
- Forward to DeepInfra with **our** key
- Record **metadata only** (account, model, token counts, cost). No prompt or completion bodies.

It does not run the agent loop, tools, or repo access. That stays in the harness on the user's machine.

## Docs

| Doc | Contents |
|-----|----------|
| [approach.md](./approach.md) | Go service shape, request path, auth, billing, ZDR, deploy |
| [boundary.md](./boundary.md) | What is OSS vs closed, and what the harness is allowed to know |

## Stack

**Go.** Same language as the harness, separate module (`api/`) in the monorepo. Details in [approach.md](./approach.md).

## Related

- [Harness providers](../harness-architecture/providers.md) — BYOK and direct calls
- [Harness tech stack](../harness-architecture/tech-stack.md) — OSS CLI
- [DeepInfra research](../research/01-deepinfra.md)
