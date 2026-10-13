# hooks

Project code that runs before or after a tool call. Hooks are files in `.thee/hooks/`, not model-written shell. The model does not call a hook by name. The runtime does, using the tool name and the arguments.

| | |
|--|--|
| **When** | `pre` can deny the call. `post` can append a short note to the tool result. |
| **Limit** | A hook gets a few seconds. Its output is capped. |
| **Permission** | Hooks run as the user, so they run only in a [trusted workspace](../security-and-policy.md#workspace-trust). A failing `pre` hook blocks the tool. |

Hooks are for formatters, secret scans, and policy. They are not a second agent. User-level hooks in `~/.thee/hooks/` always run; project hooks in `.thee/hooks/` are skipped until the workspace is trusted, and a change to any hook file asks for trust again.

## Comparison

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **thee** | Files in `.thee/hooks/`. `pre` can deny. `post` can annotate. The runtime invokes them, with a short timeout and a capped result. |  |
| **OpenCode** | Plugin and hook points surround tool calls. | closest |
| **Aider** | No hook runner. |  |
| **Ante** | Confirmation is in the daemon, not a project hook. |  |
| **Pi** | An extension can wrap a tool. |  |
| **Goose** | Extensions wrap tools instead of a hook directory. |  |
