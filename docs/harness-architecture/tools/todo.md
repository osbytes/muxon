# todo

A session task list. The model adds items, marks them done, and the director puts the open list back into the prompt so a long task does not depend on the raw transcript.

| | |
|--|--|
| **Args** | action (`add`, `done`, `list`), text |
| **Limit** | 30 open items |
| **Permission** | any tier |

`finish` is rejected while required items are still open, unless the user clears them.

## Comparison

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **thee** | A capped list, re-injected each turn. Finish waits while required work is open. |  |
| **OpenCode** | Task lists show up in OpenCode-style harnesses and their todo tools. | closest |
| **Aider** | No todo tool. The chat is the list. |  |
| **Ante** | A profile could add one. It is not the default loop. |  |
| **Pi** | A skill or extension can track tasks. |  |
| **Goose** | A recipe can carry steps. |  |
