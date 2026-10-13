# subagent

Spawn a child agent on a bounded task and return its summary, not its transcript. The child gets a fresh context and the same tool policy as the parent unless the parent narrows it.

| | |
|--|--|
| **Args** | task, kind (`read`, `write`), optional tool allow-list, optional model role |
| **Limit** | Depth 2. A child cannot spawn another child. One screen of summary comes back. |
| **Permission** | `read` child: any tier. `write` child: edit. |

## Kinds

| Kind | Use |
|------|-----|
| `read` | Exploration: "find where X is configured", "summarize how module Y handles errors". Read-only tools, no worktree. Keeps search chatter out of a small worker's context, which is the main win for frugal models. |
| `write` | Parallel or isolated edits in a [worktree](./git.md). |

## Budget

The child's context does not consume the parent's context window. The child's tokens and dollars **do** count toward the session budgets; its own cap is a slice of what remains (default 25%).

## Merge back (`write` children)

1. The child works in a worktree on branch `thee/<session>/<child>`, created from the parent's current tree.
2. The child must pass the verifier in its worktree before it can `finish`. A child that fails returns its failure summary and no diff.
3. On success the parent receives the summary and the diff stat. The runtime applies `base..child` to the parent workspace as one patch batch with `git apply --3way`.
4. Conflicts are returned to the parent as a failed tool result with the conflicting paths. Nothing is half-applied.
5. The parent's own verifier still gates the parent's `finish`. The worktree and branch are removed after merge or when the session ends.

## Comparison

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **thee** | A child with its own context. Only the summary returns. Read-only children first; writing children get a worktree and must verify before their diff is merged. |  |
| **OpenCode** | Subagents and parallel sessions are part of the product. | closest |
| **Aider** | No subagent tool. |  |
| **Ante** | The daemon can run a turn. Child agents are not the default. |  |
| **Pi** | A skill can describe a subagent. The core is one loop. |  |
| **Goose** | Recipes stay in one session. |  |
