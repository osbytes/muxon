# list

List a directory or expand a glob. Returns names, not file bodies.

| | |
|--|--|
| **Args** | path, optional glob |
| **Limit** | 200 entries |
| **Permission** | read-only |

Ignored and bulk directories are collapsed to one entry with a count, so a listing does not fill up with `node_modules`.

## Comparison

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **muxon** | A capped listing inside the workspace. Always available. One call. |  |
| **OpenCode** | Listing is part of the file tools, along with LSP. | closest |
| **Aider** | The repo map stands in for browsing the tree. |  |
| **Ante** | A profile can include or drop a list tool. |  |
| **Pi** | Listing comes from the tool pack you enable. |  |
| **Goose** | A developer extension lists files. |  |
