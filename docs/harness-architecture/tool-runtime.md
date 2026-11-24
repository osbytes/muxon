# Tool runtime

The tool runtime executes every tool in [tools/README.md](./tools/README.md). It checks permission, runs the call, truncates the result, and redacts secrets before the model sees the result.

## Rules

- A small schema per tool. The director chooses an active phase. How tools are offered depends on the model's `tool_protocol`, and whether output is constrained depends on `structured_output`. See [tools/README.md](./tools/README.md#tool-calling-protocols).
- Every reply goes through the same intake: separate reasoning, salvage in-text calls, validate against the schema, then execute or return one error ([salvage](./tools/README.md#salvage-before-repair)).
- A call identical to a recent one on unchanged files is answered from the result cache and counted for [loop detection](./session-director.md#loop-detection).
- A call outside the active phase or tier is rejected with a one-line error, not executed.
- Reads are safe. File changes go through `patch`, `write`, `delete`, and `git`.
- `shell` is not how source is saved, and it is not how verification runs. The verifier runs its own subprocess ([verifier.md](./verifier.md)).
- Streamed reply text is not written to disk.
- Each model call sets `max_tokens`. Tool arguments count as output tokens. A `write` body also counts against the file size cap.

## Permissions

| Tier | Tools |
|------|--------|
| `read-only` | search, list, read, repo-map, lsp, git status/diff/log, todo, skills, memory read/search, ask_user, subagent (read-only child), checkpoint list |
| `edit` | + patch, write, delete, git commit and worktree, memory write (project), checkpoint save/restore, test, subagent (writing child) |
| `full` | + shell, process, browser, MCP servers that write or use the network |

- `finish` is allowed in every tier. It requires a verifier pass whenever the session changed the workspace.
- Writing user-global memory (`~/.muxon/memory/`) needs approval in every tier. See [memory](./tools/memory.md).
- `test` runs repo code, so like the verifier it also requires workspace trust.
- Hooks run as the user and can deny a call before it starts. They only run in trusted workspaces.

The user's `max_permission` in `~/.muxon/config.toml` caps every session. Project config and flags can lower the tier; they cannot raise it above that cap. See [security-and-policy.md](./security-and-policy.md#config-precedence).

## Files

- **Workspace root only.** Paths are resolved and symlinks followed before the check; anything outside the root is rejected.
- **Ignore rules.** `search`, `list`, and `repo-map` skip `.gitignore` and `.muxonignore` matches and bulk directories (`node_modules`, `vendor`, `dist`, `build`, `.git`). `read` of an ignored-but-not-secret path is allowed when named explicitly.
- **Secret paths.** `read`, `search`, and `list` never return the contents of the secret deny list (`.env*` except `.env.example`, keys, credentials). Writes to those paths need approval. List in [security-and-policy.md](./security-and-policy.md#ignore-rules-and-secret-paths).
- **Stale reads.** The runtime records a content hash per file when the model reads it. `patch` and `write` on a file whose current hash differs from the last one the model saw (the user edited it meanwhile) are rejected with "file changed since you read it; re-read lines X–Y." This turns silent patch failures into one clear repair step.
- **Atomic writes.** Every write goes to a temp file in the same directory and is renamed into place.

## Subprocesses

Applies to `shell`, `process`, `test`, hooks, and the verifier.

- **argv only.** Commands are exec'd directly, never through `sh -c`. A call whose argv is a shell (`sh`, `bash`, `zsh`, `cmd`, `powershell`) with `-c` is denied unless `allow_shell_strings = true`. This keeps the blocklist meaningful.
- **Env scrubbing.** Provider keys (every `api_key_env` in the manifest and catalog) and common secret variables (`*_API_KEY`, `*_TOKEN`, `*_SECRET`, `AWS_*`) are removed from the child environment. Allowlist with `[security.env] pass = [...]`.
- **Process groups.** Each subprocess gets its own group so timeouts and interrupts kill the whole tree.
- **Sandbox** when enabled: see [security-and-policy.md](./security-and-policy.md#sandboxing).

## Results

- Every result is wrapped with its source and trust level before it enters context ([context-engine.md](./context-engine.md#untrusted-content)).
- Outputs are capped per tool and redacted with the trace redaction patterns.
- After `browser`, networked `mcp`, or `fetch` content enters the session, the runtime marks the session **tainted**; see [security-and-policy.md](./security-and-policy.md#prompt-injection) for what that gates.

## Related

- [tools/README.md](./tools/README.md)
- [security-and-policy.md](./security-and-policy.md)
- [verifier.md](./verifier.md)
- [context-engine.md](./context-engine.md)
