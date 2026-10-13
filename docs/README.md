# thee-harness — documentation

**Product:** [product.md](./product.md)

## Harness architecture (spec)

**Start here for system design:** [harness-architecture/README.md](./harness-architecture/README.md)  
**CLI (commands & TUI):** [harness-architecture/cli.md](./harness-architecture/cli.md)  
**Implementation plan:** [harness-architecture/implementation-plan.md](./harness-architecture/implementation-plan.md)  
**Stack:** [harness-architecture/tech-stack.md](./harness-architecture/tech-stack.md) (Go, open source)

## API (closed source)

**Inference front door:** [api/README.md](./api/README.md) — Go service between the harness and DeepInfra. The harness stays public; this API does not.

## Research

Foundational research for a **frugal, model-agnostic engineering harness** (CLI-first) with optional **DeepInfra** (or multi-provider) inference and **zero data retention** positioning.

| Document | Description |
|----------|-------------|
| [01-deepinfra.md](./research/01-deepinfra.md) | DeepInfra product deep dive: architecture, APIs, pricing, privacy, competitors, harness implications |
| [02-harness-landscape.md](./research/02-harness-landscape.md) | Harness landscape, techniques, influence matrix, product fit |
| [03-awesome-cli-coding-agents-survey.md](./research/03-awesome-cli-coding-agents-survey.md) | Full survey of bradAGI list (381 entries), Tier A/B for thee-harness |
| [benchmarks.md](./benchmarks.md) | How harness scores are calculated, and how to run them from this repo |

**Last updated:** 2026-10-12 (research pass).

**Primary sources:** [DeepInfra docs](https://docs.deepinfra.com/), [DeepInfra pricing](https://deepinfra.com/pricing), [Ante](https://github.com/AntigmaLabs/ante), [OpenCode](https://opencode.ai/), curated lists ([awesome-cli-coding-agents](https://github.com/bradAGI/awesome-cli-coding-agents), [awesome-agent-harness](https://github.com/xiashuqin89/awesome-agent-harness), [awesome-harness-engineering](https://github.com/ai-boost/awesome-harness-engineering)).

If you have a specific GitHub repo to fold in, add it under `docs/research/` or point the next research pass at it.
