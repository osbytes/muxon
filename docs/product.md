# Product

muxon is one coding agent harness for every model. Open or frontier, hosted or local, from any provider: you use them all from a single place, with the same loop, the same tools, and the same bill.

It is a command-line harness. You point it at a repo, describe the change, and it searches, reads, patches, and runs your tests. Which model does the work is a setting, not a different product. Swap providers or model sizes without changing how you work.

## The goal

Build the widest harness, not one tuned for a single model. muxon should drive any model from any surface (frontier APIs like Anthropic and OpenAI, aggregators like OpenRouter and DeepInfra, our own inference, and models on your own machine) through one interface.

To make that work, the harness is **optimized for the weakest model it supports**. If the loop holds up on a small open model with a short context window and shaky tool calling, it holds up everywhere. Stronger models do not need a different harness; they just finish faster and fail less. Design for the worst case, and every model benefits:

- **Tight context.** The harness picks the right files instead of sending the whole repository. That keeps small models inside their window and frontier models cheaper.
- **Reliable tool calling.** Malformed or in-text calls are caught and repaired by the harness before they cost a model turn.
- **Capability profiles.** What differs per model (tool protocol, edit format, context size, caching) comes from its profile, not from code paths written for one model class.
- **Verify before done.** No task is called done until a harness-run check passes, whichever model did the work.
- **Cost visible.** Every session shows what it spent, in tokens and dollars, so models can be compared on the same task.

A cheap model can handle most work. A larger model is used when the cheap path fails or when you choose it.

## Bring your own key, or use ours

The harness is open source, so you can bring your own key. Point it at any supported provider or a local model and the request goes straight there.

We would rather you use our inference. It is cheaper than bringing a key to the same class of model, and we do not hold your data. Zero data retention is the rule, not a setting. Prompts, completions, and code are not stored. The work still happens on your machine. Our API only passes the model call through and drops it.

The product is not a cheaper chatbot. It is one tight loop for every model: less context, reliable tools, a test run before the task is called done, and a bill you can see.
