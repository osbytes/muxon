# test

Run a targeted subset of the project's tests between edits, such as one failing test. This gives the model the useful part of a shell without arbitrary commands. It does not decide whether the task is done; only the full verify run does ([verifier.md](../verifier.md)).

| | |
|--|--|
| **Args** | filter (a test id, file path, or name pattern) |
| **Limit** | Same subprocess policy and failure bundle as the verifier. Default timeout 2 minutes. |
| **Permission** | edit, and a trusted workspace. Needs `[verify] filter_command` (for example `pytest -q {filter}` or `go test ./... -run {filter}`). |

The filter is checked against a path or test-id pattern and passed as one argv element, so it cannot add flags or commands. A passing `test` run does not set the verified flag. Without a `filter_command` the tool is not offered.

## Comparison

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **muxon** | The project's own test command with one filter. No free-form command. Never sets the verified flag. |  |
| **OpenCode** | The model runs tests through shell. |  |
| **Aider** | `/test` runs the configured command; failures go into the next turn. | closest |
| **Ante** | Tests are a shell call the daemon confirms. |  |
| **Pi** | Whatever shell tool is enabled. |  |
| **Goose** | Shell through an extension. |  |
