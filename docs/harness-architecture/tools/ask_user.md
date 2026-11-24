# ask_user

Ask the user one question when the task cannot go on without a fact only they have. Small models otherwise guess. The session goes to `blocked` (`clarification`) until the user answers.

| | |
|--|--|
| **Args** | question, optional choices (up to 5) |
| **Limit** | 3 per goal. Further calls return "decide and state your assumption." |
| **Permission** | any tier |

Headless runs have no user. The tool returns "no user is present: choose the safest assumption and state it in the finish summary," and the session continues. It is not a failure.

The question should be answerable in one line. A question the model could answer with `search` or `read` is a wasted call; the system prompt says so.

## Comparison

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **muxon** | One question, optional choices, three per goal. Headless gets an answer to proceed on its assumption. |  |
| **OpenCode** | The model asks in plain text and the turn ends. |  |
| **Aider** | The model asks in chat. |  |
| **Ante** | The daemon can pause for input. | closest |
| **Pi** | Plain-text questions. |  |
| **Goose** | The model stops and asks in chat. |  |
