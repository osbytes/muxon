# shell

Run one command in the workspace for ad-hoc debugging. It is not how source files are saved, and it is not how tests are judged: the verifier runs its own subprocess ([verifier.md](../verifier.md)), and targeted test runs go through [test](./test.md).

| | |
|--|--|
| **Args** | argv, optional cwd, timeout |
| **Limit** | 30 seconds, 64 KiB combined stdout and stderr (the model sees the exit code and the last lines; the rest is elided) |
| **Permission** | full. Off unless the user allows it. |

The argv is exec'd directly, not through a shell. `sh -c "..."` and similar are denied unless `allow_shell_strings = true`. Provider keys and secret env vars are removed from the environment. Blocked patterns include `rm -rf` and piping a download into a shell; the blocklist is a backstop, and the optional sandbox is the boundary ([security-and-policy.md](../security-and-policy.md#sandboxing)). Secrets in the output are redacted before the model sees them. After untrusted web or MCP content enters the session, a call needs approval ([taint gating](../security-and-policy.md#prompt-injection)).

## Comparison

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **muxon** | One argv, no shell strings, a timeout, an output cap, a scrubbed env, and a blocklist. Off unless the user allows it. Not used for verify. |  |
| **OpenCode** | Shell is a normal tool, with approvals, and a wider command surface. |  |
| **Aider** | Optional and usually user-triggered (`/run`). |  |
| **Ante** | The daemon confirms commands. A profile can remove shell. |  |
| **Pi** | Shell is a tool you enable. Confirmation depends on the extension. |  |
| **Goose** | Prompts before a command runs. Extensions can add more than one shell. | closest |
