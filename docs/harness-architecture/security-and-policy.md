# Security & policy

Security wraps **tool runtime**, **subprocess** execution, and **repo-supplied config**. Trust positioning: **ZDR** for inference metadata, **local control** for code execution.

## Zero data retention (product)

| Layer | Policy |
|-------|--------|
| **Our API** | Optional; no prompt/completion persistence; billing metadata only. **BYOK direct mode unchanged.** |
| **Credentials** | **BYOK** — keys never sent to muxon servers unless user opts into hosted proxy. Never passed to subprocesses. |
| **Recommended defaults** | OSS models on DeepInfra when user configures `DEEPINFRA_API_KEY`; strict mode blocks unwanted third-party model routes |
| **Local traces** | User-owned disk; optional body-free mode |

## Workspace trust

Cloning a hostile repo and running `muxon` must not be equivalent to running its scripts. Repo-supplied surfaces that can execute or steer:

| Surface | Risk | Untrusted behavior |
|---------|------|--------------------|
| `.muxon/hooks/` | Runs as the user | Not run |
| Verify command (`muxon.toml`, `package.json`, `Makefile`) | Runs as the user | Interactive: shown and approved once. Headless: setup error unless `--trust` or `--verify-cmd` |
| `test` tool | Runs repo code | Disabled |
| Project `muxon.toml` `[security]` | Could loosen policy | Can only **tighten** (lower tier, add blocks) |
| Project `muxon.toml` `[providers]` | Could redirect inference and leak code | Ignored |
| `.muxon/skills/`, `.muxon/memory/` | Prompt injection | Not loaded |
| `AGENTS.md` and source files | Prompt injection | Loaded, wrapped as untrusted content |

Trust is per workspace root and stored in `~/.muxon/trust.json` with a hash of the trust-relevant files (`muxon.toml`, `.muxon/hooks/*`, resolved verify command). When that hash changes, interactive mode asks again and shows what changed.

- **Interactive first run:** one prompt listing hooks, the verify command, and any policy the project config sets. Choices: trust, trust once, don't trust.
- **Headless / CI:** `--trust`, `MUXON_TRUST=1`, or a path prefix in the user's `trusted_roots`. CI checkouts of your own repo normally pass `--trust`.
- `muxon trust list | add | rm` manages the store.

## Config precedence

Lowest to highest:

1. Built-in defaults
2. User `~/.muxon/config.toml`
3. Project `muxon.toml` (only its tightening keys when untrusted)
4. Environment `MUXON_*`
5. CLI flags

Security keys are the exception: the user's `max_permission`, `deny` patterns, and `zdr_strict` are ceilings that later layers cannot loosen. `--yolo` raises the tier to the user's `max_permission` and auto-approves within it; it never exceeds it.

## Permission tiers

| Tier | Capabilities |
|------|----------------|
| `read-only` | search, list, read, repo-map, lsp, git read, memory read/search, ask_user, read-only subagent |
| `edit` | + patch, write, delete, git write, checkpoint, project memory write, test, writing subagent |
| `full` | + shell, process, browser, networked MCP |

Full list and per-tool notes: [tool-runtime.md](./tool-runtime.md#permissions).

Interactive prompts on first `shell` or destructive patch in `edit` mode (Smelt/Codex-inspired).

## Prompt injection

Untrusted content enters context through `read`, `search`, `browser`, MCP, subprocess output, and memory. Small workers are easier to steer, so the harness does not rely on the model resisting.

**Wrapping and fixed policy:**

- Every tool result is wrapped with source and trust level; the system prompt has one fixed rule that tool content is data ([context-engine.md](./context-engine.md#untrusted-content)).
- Policy never comes from content: tier, verify command, approvals, and provider routing cannot be changed by anything the model reads.

**Taint gating:**

Once `browser`, networked `mcp`, or any fetched URL content has entered the session, the session is **tainted** until the next user message. While tainted:

| Action | Interactive | Headless |
|--------|-------------|----------|
| `shell`, `process`, networked `mcp` | Approval required | Denied |
| `browser` to a new host | Approval required | Denied |
| `git commit`, user-global memory write | Approval required | Denied |
| `read`, `patch`, `write`, `test` | Normal | Normal |

The user's next message clears the taint because the user has seen the content in the transcript.

## Shell guardrails

- **argv only;** `sh -c` style strings denied unless `allow_shell_strings = true`.
- Blocklist: `rm -rf /`, `curl | sh`, fork bombs, writing outside workspace root. The blocklist is a backstop, not a boundary; the sandbox is the boundary.
- Allowlist optional in `muxon.toml` for CI.
- Provider keys and secret env vars scrubbed from every subprocess.
- Secret redaction in trace: patterns for API keys, `.env` paths.

## Sandboxing

| Layer | Scope |
|-------|-------|
| **Always on** | argv-only exec, env scrubbing, process-group kill, workspace path checks |
| **OS sandbox** | For `shell`, `process`, `test`, and the verifier: write access to the workspace and temp only, network off unless `[sandbox] network = true`. On by default for the `full` tier where a backend exists; opt-in for `edit`. |
| **Egress allowlist** | Per-command network allowlist when sandbox network is on (DvalinCode-style) |

Backends: macOS `sandbox-exec` profile; Linux bubblewrap, falling back to Landlock; **Windows has no native backend**, so `full` tier there prints a warning on every session and recommends WSL or a container. Any platform can use `[sandbox] backend = "docker"` to run subprocesses in a container with the workspace mounted, which is also the recommended CI setup.

## Ignore rules and secret paths

Prevention, not just redaction:

- `search`, `list`, and `repo-map` respect `.gitignore`, `.muxonignore` (gitignore syntax), and skip bulk directories (`node_modules`, `vendor`, `dist`, `build`, `target`, `.git`, `.venv`).
- **Secret deny list** — contents never returned by `read`/`search`, writes need approval:

```text
.env .env.* (except .env.example, .env.sample, .env.template)
*.pem *.key *.p12 *.pfx id_rsa* id_ed25519* *.keystore
.aws/** .ssh/** .gnupg/** .netrc .npmrc .pypirc
**/credentials* **/secrets.* *.tfstate
```

Users extend it with `[security] deny_read = [...]`. A denied read returns "path is on the secret deny list", so the model stops asking rather than guessing.

## Memory poisoning

Memory is injected into future sessions, so a poisoned note outlives the session that wrote it.

- Every note stores provenance: session id, the tool results it followed, timestamp, and whether the session was tainted.
- Project memory (`.muxon/memory/`) is written at `edit`; it is loaded only in trusted workspaces.
- User-global memory (`~/.muxon/memory/`) writes always need approval (headless: denied), because they affect every repo.
- Notes written while tainted are stored as `pending` and not injected until the user accepts them (`muxon memory review`).
- `muxon memory list | show | rm` for inspection.

## Network egress

DvalinCode-style:

- Per-request egress check for subprocesses.
- Enforced by the sandbox layer above.

## Audit trail (team policy)

- Hash-chained log of tool calls and approvals for compliance export.
- Off for solo use; on when a team policy requires it.

## Config surface: `muxon.toml`

```toml
[providers]                          # honored only in ~/.muxon/config.toml
catalog = "~/.muxon/catalog.json"     # BYOK provider registry (optional)
models_manifest = "models.yaml"      # or path to user override

[security]
permission = "edit"
zdr_strict = true
allow_shell = false
allow_shell_strings = false
deny_read = ["config/master.key"]

[security.shell]
block_patterns = ["rm -rf", "curl.*\\|.*sh"]

[security.env]
pass = ["PATH", "HOME", "GOPATH"]

[sandbox]
enabled = "auto"     # auto (on for full tier) | true | false
backend = "auto"     # auto | sandbox-exec | bubblewrap | landlock | docker
network = false
```

User-level `~/.muxon/config.toml` adds `max_permission` and `trusted_roots`.

## Related

- [tool-runtime.md](./tool-runtime.md)  
- [providers.md](./providers.md) — ZDR model gating  
- [observability.md](./observability.md) — redaction in traces  
- [verifier.md](./verifier.md) — verify command trust
