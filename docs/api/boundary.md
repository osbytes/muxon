# OSS harness / closed API boundary

## Modules (monorepo)

| Module | Path | Visibility | Language |
|--------|------|------------|----------|
| Harness | `harness/` | Public | Go |
| API | `api/` | Private | Go |

They share **no internal packages**. The contract is HTTP.

## Contract the harness may rely on

- `POST /v1/chat/completions` with streaming
- `GET /v1/models`
- Bearer muxon API key
- OpenAI-style `usage` on the final stream chunk (prompt, completion, cached, and reasoning tokens)
- Request fields passed through unchanged: `tools`, `tool_choice`, `response_format` (including `json_schema`), `prompt_cache_key`, sampling parameters, `stop`, and reasoning parameters
- Response fields passed through unchanged: `tool_calls`, `reasoning_content`, `finish_reason`
- A capability block per model on `GET /v1/models`, in the same schema as the harness manifest ([capability fields](../harness-architecture/providers.md#capability-fields))
- Error JSON with HTTP status codes from [approach.md](./approach.md)

If the private API changes a field, it keeps the OpenAI shape so the public client does not fork.

## What stays out of the public repo

- DeepInfra key and any other upstream secret
- Account database, billing, key issuance
- Margin and wholesale terms
- Abuse heuristics beyond documented status codes

## What stays out of the private API

- Workspace files, shell, git, verifier
- System prompts, skills, compaction
- A requirement that users must use our API

A documented `muxon` provider in the public manifest is fine. Hard-coding a private URL with an embedded secret is not.

## Default user experience

1. Install the public CLI.
2. Either `muxon auth login` (muxon API key) or set a BYOK key in `catalog.json`.
3. The loop is the same either way.

## Related

- [approach.md](./approach.md)
- [README.md](./README.md)
