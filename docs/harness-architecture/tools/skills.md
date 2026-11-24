# skills

Load one procedure by name. The prompt holds a catalog of names and one-line descriptions. The body of a skill is injected only after this tool runs, and at most two bodies at a time.

| | |
|--|--|
| **Args** | name |
| **Limit** | Two loaded skills. Bodies count toward the context budget. |
| **Permission** | any tier |

Skills live in the repo (`.muxon/skills/`) and in the user directory (`~/.muxon/skills/`). A skill is instructions. It does not add a new executable tool. Repo skills appear in the catalog only in a [trusted workspace](../security-and-policy.md#workspace-trust); user skills always do. A skill cannot change permissions or the verify command.

## Comparison

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **muxon** | A catalog of names in the prompt. The body loads on demand, two at a time. Instructions only. |  |
| **OpenCode** | Skills are files the agent can load, often next to MCP. |  |
| **Aider** | No skill catalog. Conventions stay in one file. |  |
| **Ante** | A profile swaps prompt, tools, skills, and memory together. |  |
| **Pi** | Skills are a primary extension surface. | closest |
| **Goose** | A recipe can also turn tools on. |  |
