# lsp

Ask the language server for a definition, references, or diagnostics. The result is a short list of locations, which the model then `read`s.

| | |
|--|--|
| **Args** | action (`definition`, `references`, `diagnostics`), path, optional line |
| **Limit** | 30 locations |
| **Permission** | read-only |

The harness starts the server for the project’s languages. If none is installed, the tool returns that fact and the model uses `search`.

## Comparison

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **muxon** | Definition, references, and diagnostics, capped at 30 locations, when a server exists. |  |
| **OpenCode** | LSP is loaded automatically and is part of the main tool set. | closest |
| **Aider** | No LSP. The repo map is the index. |  |
| **Ante** | Not a required part of the daemon. |  |
| **Pi** | Editor features come from extensions. |  |
| **Goose** | An extension can wrap a language server. |  |
