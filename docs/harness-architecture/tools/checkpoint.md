# checkpoint

Save the workspace and restore it. A checkpoint is a snapshot of the tree plus the session’s todo and memory pointers. Restore puts files back and reopens the session at that turn.

| | |
|--|--|
| **Args** | action (`save`, `restore`, `list`), optional id |
| **Limit** | The last 20 checkpoints are kept for the session. |
| **Permission** | `list` is read-only. `save` and `restore` are edit. |

## Storage

- **Git repo.** Checkpoints are **shadow refs** under `refs/thee/sessions/<id>/cp-<n>`. The harness builds them with a temporary index (`GIT_INDEX_FILE`), `git write-tree`, and `git commit-tree`, so the user's index, `HEAD`, branches, and stash are never touched and `git log` does not show them. Untracked non-ignored files are included. Refs older than 14 days are pruned by `thee gc`.
- **Not a git repo.** Changed files are copied to `.thee/sessions/<id>/snapshots/<n>/` before each patch batch (copy-on-write for files the agent touched only).

The session base (`refs/thee/sessions/<id>/base`) is taken at start, including a dirty tree, so restore can always get back to exactly what the user had.

Restore does not rewind our API. It only rewinds the local workspace. The model can call `restore` after a bad patch. The user can call it from the CLI.

## Comparison

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **thee** | Shadow git refs (or file snapshots) plus session pointers. The last 20 are kept. User history is untouched. Restore is local. |  |
| **OpenCode** | Sessions can be resumed. Workspace undo is lighter. |  |
| **Aider** | Git history is the undo path. | closest |
| **Ante** | The daemon keeps the turn. Workspace checkpoints are not the core. |  |
| **Pi** | An extension can snapshot. |  |
| **Goose** | Session state is kept. File undo is not the product. |  |
