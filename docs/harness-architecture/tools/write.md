# write

Create a file, or replace a small one. Parent directories may be created under the workspace. A path outside the workspace is rejected.

| | |
|--|--|
| **Args** | path, contents |
| **Limit** | 32 KiB. A larger existing file must be `patch`ed, not overwritten. |
| **Permission** | edit |

Replacing an existing file the user changed since the model last read it is rejected, the same as `patch`. Writes to secret-deny-list paths need approval. The tool returns the path and a short diff stat. Reply text streamed to the terminal is not saved. The body of this call counts as output tokens and against the file cap. See [tool runtime](../tool-runtime.md).

## Comparison

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **muxon** | Full contents for a new or small file, capped at 32 KiB. A large existing file is patched. |  |
| **OpenCode** | A write can drop a large new file in one call. | closest |
| **Aider** | New files usually arrive as a diff. |  |
| **Ante** | A profile can allow write. Size limits are not part of the public design. |  |
| **Pi** | Write is a standard tool without that cap. |  |
| **Goose** | Write lives in an extension and can be as large as the model emits. |  |