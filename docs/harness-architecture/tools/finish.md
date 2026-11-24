# finish

Tell the session the goal is done. If the session changed the workspace, the runtime accepts the call only after the verifier passes on the current tree. If nothing changed (a `question` goal), `finish` is accepted directly and the summary must cite `path:line` for its claims.

| | |
|--|--|
| **Args** | short summary, including any assumptions made after an unanswered `ask_user` |
| **Limit** | One successful call ends the goal |
| **Permission** | any tier |

Calling `finish` before a pass makes the harness run verify. A failed verifier sends the model back to work, and the caps in the [session director](../session-director.md#caps-and-budgets) decide when it stops as `failed` instead. An edit after a pass clears it. `finish` is also rejected while required [todo](./todo.md) items are open.

The summary the user sees adds the diff stat against the session base and lists any protected test files changed ([verifier.md](../verifier.md#verify-gaming)).

## Comparison

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **muxon** | Finish is rejected until the verifier has passed, unless nothing was changed. |  |
| **OpenCode** | The model stops when it says it is done. Tests are optional. |  |
| **Aider** | You end the turn. `/test` does not block a finish tool. |  |
| **Ante** | The loop ends on the model or the user. |  |
| **Pi** | No finish gate. |  |
| **Goose** | The recipe ends when the model stops calling tools. |  |
