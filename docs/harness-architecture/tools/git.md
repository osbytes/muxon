# git

Status, diff, commit, and worktree. Day-to-day git does not go through a free-form shell string.

| | |
|--|--|
| **Args** | action (`status`, `diff`, `log`, `commit`, `worktree`), optional path and message |
| **Limit** | Diff output is capped and truncated like `shell` |
| **Permission** | `status`, `diff`, and `log` are read-only. `commit` and `worktree` are edit. |

`commit` requires a message and refuses a commit that includes secrets. It commits only to the current branch, never pushes, and in headless runs is off unless `[git] allow_commit = true`. `worktree` creates an isolated checkout for a [subagent](./subagent.md). Checkpoints use shadow refs and do not go through this tool ([checkpoint](./checkpoint.md)).

## Comparison

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **muxon** | Named actions. Status, diff, and log are read-only. Commit and worktree are edits. |  |
| **OpenCode** | Git happens through the shell and the editor. |  |
| **Aider** | Git commits and diffs are the center of the edit loop. | closest |
| **Ante** | A profile can allow shell git instead. |  |
| **Pi** | Git is a tool you add. |  |
| **Goose** | An extension wraps git. |  |
