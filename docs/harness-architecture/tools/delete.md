# delete

Delete a file or move it. Both stay inside the workspace. A directory delete has to be empty, unless the user has already approved a recursive delete.

| | |
|--|--|
| **Args** | path, and for a move, the destination |
| **Limit** | One path per call |
| **Permission** | edit |

The tool returns the path and whether it was removed or renamed. It does not dump the old body into context.

## Comparison

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **thee** | A delete and a move, one path per call. |  |
| **OpenCode** | File tools include remove and rename. | closest |
| **Aider** | Removal is usually a diff that deletes every line, or a git command. |  |
| **Ante** | A profile decides if delete exists. |  |
| **Pi** | Delete is whatever the tool pack includes. |  |
| **Goose** | An extension deletes files. |  |
