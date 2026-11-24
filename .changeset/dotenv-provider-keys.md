---
"muxon": minor
---

`muxon` reads provider keys such as `OPENCODE_API_KEY` from the nearest `.env` at or above the current directory, up to the repo root. Variables already set in your shell win over the file. Copy `.env.example` to `.env` to get started.

**Upgrade:** a `.env` line that isn't `KEY=VALUE` (for example `name: value`) now stops every `muxon` command with the file and line number. Fix or remove that line.
