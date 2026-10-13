# read

Read one slice of one file. The model asks for a line range after `search`, instead of pasting the file into context.

| | |
|--|--|
| **Args** | path, start_line, end_line |
| **Limit** | 400 lines per call |
| **Permission** | read-only |

Paths must stay inside the workspace. A miss returns an error, not a guess. Paths on the secret deny list (`.env`, keys, credentials) are refused; ignored paths such as `node_modules/` are readable only when named explicitly ([security-and-policy.md](../security-and-policy.md#ignore-rules-and-secret-paths)). The runtime records a content hash so a later `patch` can detect that the user changed the file.

## Comparison

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **thee** | One file, one line range, 400 lines max. The model chooses the range. |  |
| **OpenCode** | Reads files and can pull symbols through the LSP. |  |
| **Aider** | You name the files. It reads those. |  |
| **Ante** | File read is a profile tool. Caps depend on the profile. |  |
| **Pi** | Read is a basic tool. No published line cap. | closest |
| **Goose** | Read is an extension. Output limits are looser. |  |