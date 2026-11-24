---
"muxon": patch
---

Eval tasks, hidden checks, fixtures and model matrices moved from `harness/testdata/eval` to a root `evals/` folder. With no `--tasks`, `muxon eval basket` now reads `evals/tasks/*.json`. Matrices are `evals/matrices/{default,ci,opencode}.yaml`.

**Upgrade:** scripts that pass `--matrix testdata/eval/...` or `--tasks 'testdata/eval/*.json'` need the new paths.
