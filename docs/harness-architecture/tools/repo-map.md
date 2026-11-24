# repo-map

Return a map of symbols in the repo: file, name, and line. It is built from the tree, not by pasting source into the prompt. The model uses it to decide what to `search` or `read`.

| | |
|--|--|
| **Args** | optional path or language filter |
| **Limit** | Fits in the context budget. Ranked by relevance to the task, not alphabetical dump. |
| **Permission** | read-only |

The map skips ignored, bulk, and secret paths. It is rebuilt when git HEAD or the dirty set changes. It sits in the slow-changing part of the prompt so it can stay in the cache prefix.

## Comparison

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **muxon** | A ranked symbol map, always on. The model still reads the bodies it edits. |  |
| **OpenCode** | LSP and file tools, without Aider’s map as the index. |  |
| **Aider** | The repo map is how the agent orients before editing. | closest |
| **Ante** | A profile can skip it. |  |
| **Pi** | Indexing is an extension if you add one. |  |
| **Goose** | An extension can index the repo. |  |
