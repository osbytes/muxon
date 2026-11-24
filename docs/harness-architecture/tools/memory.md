# memory

Read and write notes that survive the session. Memory is on the user’s machine. Our API does not store it.

| | |
|--|--|
| **Args** | action (`read`, `write`, `search`), scope (`project`, `user`), text or query |
| **Limit** | A note is at most 2 KiB. Search returns a handful of hits. |
| **Permission** | `read` and `search`: any tier, including `read-only`. `write` to project memory (`.muxon/memory/`): `edit`. `write` to user memory (`~/.muxon/memory/`): approval in every tier; denied headless. |

Small models get cheap structured recall without patch rights. Gating writes keeps read-only runs side-effect-free (apart from the trace) and stops a skim of an untrusted repo from planting notes.

Project memory is loaded only in a trusted workspace, because it is repo-supplied text. Every note records provenance (session, the tool results it followed, time). A note written after untrusted web or MCP content entered the session is kept `pending` until the user accepts it with `muxon memory review`. See [security-and-policy.md](../security-and-policy.md#memory-poisoning).

The context engine injects the search hits, not the whole store, wrapped as model-written content. Memory is for conventions and decisions, not a copy of the repo.

## Comparison

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **muxon** | Local notes with explicit read, write, and search. Size-capped, with provenance. Reads in any tier, writes from `edit`, user-global writes approved. |  |
| **OpenCode** | Session history is the memory. |  |
| **Aider** | The repo and conventions file are the memory. |  |
| **Ante** | A profile includes memory beside the prompt and tools. | closest |
| **Pi** | Memory is something an extension can add. |  |
| **Goose** | Session state, plus extensions. |  |
