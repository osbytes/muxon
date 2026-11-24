# search

Find matching lines in the workspace. Returns file, line number, and a short preview. It does not return whole files.

| | |
|--|--|
| **Args** | pattern, optional path, glob, max_hits |
| **Limit** | 50 hits, 200 characters of preview each |
| **Permission** | read-only |

Implementation wraps `rg` when it is on `PATH`. It respects `.gitignore` and `.muxonignore`, skips bulk directories (`node_modules`, `vendor`, `dist`, …), and never returns lines from paths on the secret deny list ([security-and-policy.md](../security-and-policy.md#ignore-rules-and-secret-paths)).

## Comparison

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **muxon** | Ripgrep with a hit cap and a short preview. Always on. The model greps, then reads a range. |  |
| **OpenCode** | Search plus LSP symbol lookup. |  |
| **Aider** | Repo map, then you add files. |  |
| **Ante** | Grep is a normal tool. A profile can omit it. | closest |
| **Pi** | Search comes from whatever tools or skills you enable. |  |
| **Goose** | Search is an extension and often broader. |  |