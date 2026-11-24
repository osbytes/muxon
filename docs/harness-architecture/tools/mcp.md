# mcp

Call a tool on an MCP server the user has attached. The server’s tools are not all listed in the prompt. The model sees server names and tool names, and this call runs one of them.

| | |
|--|--|
| **Args** | server, tool, arguments |
| **Limit** | The result is truncated like `shell` output. |
| **Permission** | The server declares a tier. A server that can write or reach the network is full. |

Config lives in `muxon.toml`. A server that fails to start is reported and skipped. MCP does not replace `patch`, `read`, or `shell`. Servers declared in a project `muxon.toml` start only in a trusted workspace. Results are untrusted content, and a networked server's result taints the session ([security-and-policy.md](../security-and-policy.md#prompt-injection)).

## Comparison

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **muxon** | One call to a user-attached server. The result is truncated. Built-in edit tools stay. A server that writes is full permission. |  |
| **OpenCode** | MCP is supported beside a large built-in tool set. |  |
| **Aider** | No MCP. |  |
| **Ante** | Extensions are profiles more than MCP. |  |
| **Pi** | MCP is a core extension path. |  |
| **Goose** | MCP is how the agent grows tools. | closest |
