# Product

thee is a coding agent for engineers who are tired of paying frontier-model prices for work a smaller model can do, if it is given the right files instead of the whole repository.

It is a command-line harness. You point it at a repo, describe the change, and it searches, reads, patches, and runs your tests. The default models are small open ones, good enough for most of the work. A larger model is used only when the cheap path fails. Every session shows what it spent, in tokens and dollars.

The harness is open source, so you can bring your own key. Point it at DeepInfra, OpenRouter, or a model on your own machine and the request goes straight there.

We would rather you use our inference. It is cheaper than bringing a key to the same class of model, and we do not hold your data. Zero data retention is the rule, not a setting. Prompts, completions, and code are not stored. The work still happens on your machine. Our API only passes the model call through and drops it.

The product is not a cheaper chatbot. It is a tighter loop: less context, smaller models, a test run before the task is called done, and a bill you can see.
