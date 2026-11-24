# Evals

Self-graded tasks for `muxon eval`. The runner lives in [`harness/internal/eval`](../harness/internal/eval).

| Path | Contents |
|------|----------|
| `tasks/*.json` | One task per file: prompt, fixture, hidden check, timeout, tags |
| `checks/<task-id>/` | Hidden grader files, copied into the workspace only after the agent exits |
| `fixtures/<name>/` | Mini repos each task starts from |
| `matrices/*.yaml` | Model matrices for `eval basket` (`default`, `ci`, `opencode`) |

## Run

From the repo root:

```sh
make eval
```

Or directly, from `harness/`:

```sh
go run ./cmd/muxon eval basket --matrix ../evals/matrices/default.yaml --tasks '../evals/tasks/*.json' --trials 3 --budget-usd 20
```

With no `--tasks`, the CLI looks for `evals/tasks/*.json` from the repo root or `harness/`. Results go to `.muxon/eval/runs/<run-id>/`.

## Add a task

1. Add a fixture under `fixtures/<name>/`, or reuse one.
2. Add `checks/<task-id>/` with the hidden test or script.
3. Add `tasks/<name>.json`. `fixture` and `check.files` resolve relative to `tasks/`, so use `"fixture": "<name>"` and `"files": "checks/<task-id>"`.

`fixtures/go-failing-test` is a copy of `harness/testdata/fixtures/go-failing-test`, which the Go unit tests use. Edit them separately.
