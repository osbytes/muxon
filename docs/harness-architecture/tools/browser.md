# browser

Fetch a URL, or drive a page when the task needs a site. Page text comes back trimmed. Screenshots are a path, not a blob in the prompt, unless the model is on a vision role.

| | |
|--|--|
| **Args** | action (`fetch`, `open`, `click`, `type`, `read`), url or selector |
| **Limit** | Fetch returns at most 16 KiB of text. A browse session is a handful of steps. |
| **Permission** | full |

`fetch` is for docs and issues. Clicking and typing are for when a page has no plain URL for the answer. Prefer `fetch`.

Page content is untrusted. It is wrapped as such in context, and it **taints** the session: until the next user message, `shell`, `process`, networked MCP, commits, and new hosts need approval (denied headless). See [security-and-policy.md](../security-and-policy.md#prompt-injection). Screenshots go to the model as images only when the bound model's profile has `vision: true`.

## Comparison

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **thee** | Fetch first. A short session only when the page requires it. Text is capped. |  |
| **OpenCode** | Browsing is not the core of the editor. |  |
| **Aider** | No browser. |  |
| **Ante** | Not part of the default daemon profile. |  |
| **Pi** | A skill or MCP server can add one. |  |
| **Goose** | A browser shows up as an extension. | closest |
