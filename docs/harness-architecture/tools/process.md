# process

Start a process that outlives one tool call, then read its output later. This is for dev servers and watchers. Tests that should finish stay on [shell](./shell.md).

| | |
|--|--|
| **Args** | action (`start`, `read`, `stop`), argv, optional name |
| **Limit** | A read returns the last 16 KiB. Idle processes stop at the end of the session. |
| **Permission** | full |

A process is killed when the session ends unless the user asked to leave it. Output is redacted the same way as shell output. The same argv-only, env-scrubbing, and sandbox rules as `shell` apply ([tool-runtime.md](../tool-runtime.md#subprocesses)).

## Comparison

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **thee** | Named start, read, and stop. Log reads are capped. The session stops leftovers. Tests stay on shell. |  |
| **OpenCode** | Long commands are usually a shell call. |  |
| **Aider** | `/run` is one-shot. |  |
| **Ante** | The daemon can hold a turn. Background processes are not the published model. |  |
| **Pi** | An extension can manage one. |  |
| **Goose** | Extensions run long jobs beside the session. | closest |
