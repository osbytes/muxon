# Benchmarks

How coding-harness scores are actually produced, and how thee can report them from this repo.

A published score is a **harness–model pair** on a **pinned dataset**, graded by a program that runs after the agent exits. The model does not grade itself. The harness does not get to decide that a task passed. Holding the model fixed and swapping the harness moves the score, sometimes by more than swapping a nearby model. That is the measurement this product needs.

## What gets scored

Every serious bench has the same four pieces.

| Piece | Role |
|-------|------|
| Task | A repo or container, plus a natural-language instruction |
| Agent run | The harness and one pinned model, until it exits or the timeout fires |
| Hidden check | Tests the agent did not write, run after the agent process is gone |
| Aggregate | Passed tasks divided by tasks attempted, plus cost and time |

A task with 9 of 10 tests passing scores **0**. Partial credit exists inside some graders and is not the headline number.

Timeouts, crashes, empty patches, and harness exceptions count as failures in a fair report. Dropping them and dividing only by the tasks that “finished cleanly” inflates the rate.

## SWE-bench

[SWE-bench](https://www.swebench.com/) turns a real GitHub issue into a patch task. The canonical sets:

| Set | Size | What it is |
|-----|------|------------|
| Full | 2,294 | The original test split |
| Lite | 300 | A smaller slice used for iteration |
| Verified | 500 | Human-filtered instances (OpenAI, 2024) |

The agent sees the issue text and the repo at the base commit. It emits a unified diff. It does not see the tests that were added in the fixing pull request.

Grading is a separate Python harness, usually Docker, one container per instance ([evaluation guide](https://www.swebench.com/SWE-bench/guides/evaluation/)):

1. Start from the base commit.
2. Apply the model patch with `git apply` / `patch`. If it does not apply, the instance errors and is not resolved.
3. Apply the official `test_patch` (the hidden tests).
4. Run two lists recorded from the original PR:
   - **FAIL_TO_PASS** — tests that failed before the gold fix and passed after.
   - **PASS_TO_PASS** — tests that passed before and after, so the patch must not break them.
5. `f2p` is the fraction of FAIL_TO_PASS tests that pass. `p2p` is the fraction of PASS_TO_PASS tests that pass.

The grader’s status function is explicit (`swebench/harness/grading.py`):

- **FULL** when `f2p == 1` and `p2p == 1`. This is the only status that counts as resolved.
- **PARTIAL** when some but not all FAIL_TO_PASS tests pass and every PASS_TO_PASS test passes. Leaderboards do not give this credit.
- **NO** otherwise.

**Resolution rate** = instances resolved / instances submitted. An empty patch is submitted and unresolved. An instance that never produced a `report.json` (container failed, patch did not apply, timeout) is an error, not a quiet omission. The FAQ counts those separately from “unresolved,” and a honest report shows both. Reusing a `run_id` for the same `instance_id` returns the cached grade even if the diff changed. A new diff needs a new `run_id`.

Two known holes in the number:

- The executed tests are the tests touched by the original PR, not the repo’s full suite. A patch can be marked resolved and still fail tests that were never run ([SWE-bench #280](https://github.com/swe-bench/SWE-bench/issues/280)).
- A model patch that creates a file the hidden `test_patch` also creates can poison the grade. Upstream has patched this ([SWE-bench #538](https://github.com/swe-bench/SWE-bench/issues/538)); a local harness and the hosted `sb-cli` have disagreed. Use a current grader.

Verified is the set people quote. It is also contaminated by now, and some instances have brittle tests (a Django loader bug zeroed correct fixes). Treat Verified as a comparison set with the grader version pinned, not as a clean measure of “can this model code.”

The prediction file thee would emit is JSONL, one object per task:

```json
{"instance_id": "sympy__sympy-20590", "model_name_or_path": "thee+Qwen/Qwen3-Coder", "model_patch": "diff --git a/..."}
```

## What changes when the harness changes

Most leaderboards vary the model and leave the agent unstated. The useful experiment for thee is the opposite.

Ante ran one model (`deepseek/deepseek-v4-flash-0731`), the same ten Terminal-Bench 2.1 tasks, one trial, and the same Daytona sandbox through Ante, Ante with a short prompt, Pi 0.73.1, OpenCode 1.18.4, and Hermes ([How much does the agent harness matter?](https://antigma.ai/blog/2026/08/04/harness-matter), 4 Aug 2026). Pass counts were 10, 9, 7, 7, and 7. They priced every harness on one token table so a provider’s markup would not look like a harness cost. On that table, Pi’s invoice was about ten times the reference price of the same tokens. OpenCode’s reported cost missed reasoning tokens. Ten tasks move ten points when one task flips, so Ante correctly calls this a probe, not a ranking.

A later write-up ran GLM-5.2 through four harnesses on all 89 tasks, five trials, and reported a 13 point gap between the best and worst harness on that model ([kimchi.dev](https://kimchi.dev/blog/same-model-different-harness-terminal-bench)).

The scaffold-effect paper measured three harnesses, two models, and 50 Terminal-Bench Pro tasks (300 trials). Pass-rate gaps stayed inside 0–8 points. Tokens per solved task differed by about 40×. The Terminal-Bench paper’s own Claude Opus 4.5 pair was 52.1% on one harness and 57.8% on another, at 256.9M versus 3.9M input tokens ([paper PDF](https://kdd-eval-workshop.github.io/agenticai-evaluation-kdd2026/assets/papers/74_The_Scaffold_Effect_in_Codi.pdf)). A small accuracy gain can hide a large bill. thee’s report has to show both.

## Can this live in the repo?

Yes. The grader stays a separate program. The repo owns the runner, the adapter, and the report. The Go CLI does not reimplement pytest or Docker.

Three layers, in the order they should be built.

### 1. A basket thee grades itself

This is the number we can run on every change. It matches the eval hook already sketched in [observability](./harness-architecture/observability.md).

A task file names a fixture repo, an instruction, and a verify command. `thee eval` runs the session headless, then runs that command. Exit 0 writes reward 1. The agent does not see the verify command’s source and cannot write the result file.

Each row in `.thee/eval/results.jsonl` carries:

| Field | Why |
|-------|-----|
| `task_id`, `trial` | Identity |
| `reward` | 0 or 1, from the verify command |
| `model_id`, `provider`, `harness_sha` | The pair being measured |
| `input_tokens`, `output_tokens`, `cached_tokens` | Usage from the provider |
| `cost_usd` | Pinned list price in `models.yaml`, same table for every harness we compare |
| `wall_s`, `tool_rounds` | Speed and loop length |
| `escalations`, `malformed_tool_calls` | Whether the small-model protocol held |
| `salvaged_tool_calls`, `loops_detected` | Whether the model only worked because the harness caught its slips |
| `timeout_hit` | A timeout is a failure |

The basket from the landscape note is the right first set, held fixed: a failing unit test, a small feature with existing tests, a two-file lint fix, a dependency bump, a rename. Five tasks, several trials. This proves the loop. It is not a public leaderboard score.

Report, from those rows:

```text
pass_rate        = mean(reward)
cost_per_pass    = sum(cost_usd) / sum(reward)     # undefined if nothing passed
tokens_per_pass  = sum(input + output) / sum(reward)
seconds_per_task = mean(wall_s)
escalation_rate  = mean(escalations > 0)
malformed_rate   = sum(malformed_tool_calls) / sum(tool_rounds)
salvage_rate     = sum(salvaged_tool_calls) / sum(tool_rounds)
loop_rate        = mean(loops_detected > 0)
```

Run the basket across the model matrix in [observability](./harness-architecture/observability.md#eval-integration), smallest models included.

The task file is JSON. `check.files` is copied into the workspace only after the agent exits, and `check.cmd` is the grader. `verify_cmd` is still the command the agent is allowed to see.

```json
{"id":"fix-test-001","prompt":"fix the failing test","fixture":"go-failing-test",
 "check":{"cmd":["go","test","."],"files":"checks/fix-test-001"},
 "timeout_s":600,"tags":["go","fix"]}
```

Commands:

```text
thee eval basket --matrix harness/testdata/eval/matrix.yaml --trials 3 --budget-usd 20
make eval
thee eval report <run-id>
thee eval compare <run-a> <run-b>
thee eval export-swebench --instance repo__repo-1 --model thee+qwen3-32b -o predictions.jsonl
```

Each run lives in `.thee/eval/runs/<run-id>/` (`run.json` plus `results.jsonl`). Rows also carry `run_id`, `provider`, `harness_sha`, `manifest_hash`, `salvaged_tool_calls`, `timeout_hit`, `reason`, and `overrides`. Prices in `models.yaml` are pinned 2026-10-14 (DeepInfra Qwen3.5-9B and Qwen3-32B, OpenAI gpt-5). `matrix-opencode.yaml` uses the OpenCode Go key: `OPENCODE_API_KEY`, or `~/.local/share/opencode/auth.json` under provider `opencode-go`. Its prices are the short-context rates from the OpenCode Go docs (Kimi K2.7 Code, GPT 6 Luna).

`thee eval compare` is the gate for a new tool or harness flag: same tasks, one override flipped, and the McNemar result says whether the outcomes actually moved.

Compare against one fat-context frontier run on the same tasks, with the same price table. That is the 3× cost claim, and it only counts when pass rate stays in the same band.

### 2. Public benches through their own graders

Do not re-grade SWE-bench inside Go.

**SWE-bench.** Headless thee writes `predictions.jsonl`. Then:

```text
python -m swebench.harness.run_evaluation \
  --dataset_name princeton-nlp/SWE-bench_Lite \
  --predictions_path predictions.jsonl \
  --run_id <new id>
```

`thee eval export-swebench` writes that predictions line from `git diff HEAD` (including new files). Lite (300) is the iteration set. Verified (500) is what a public claim has to name, with the grader commit pinned. Full 2,294 is optional. Docker is required. A fresh `run_id` is required whenever the patch changes.

Traces stay on the machine. A published job lists task ids, rewards, token counts, and durations. Prompt and completion bodies stay in the local JSONL unless someone explicitly exports them.

## What a thee report contains

One block, every time:

- Dataset and version (`SWE-bench_Lite` or `thee-basket`).
- Harness git SHA and model id. The score is the pair.
- Tasks, trials, and the formula (`mean reward`, not pass@k).
- Pass rate and an interval. For public sets, bootstrap by task.
- Dollars per pass and tokens per pass, at the pinned list price, plus the provider invoice if it differs.
- Seconds per task, tool rounds, escalation rate, malformed tool-call rate.
- Whether timeouts and resource limits were left alone.
- A link to the run log.

A 10-task probe can show that the harness moves the result. It cannot be placed next to an 89-task leaderboard row.

## Sources

- SWE-bench evaluation guide: https://www.swebench.com/SWE-bench/guides/evaluation/
- SWE-bench FAQ (resolution rate): https://www.swebench.com/SWE-bench/faq/
- Grader: https://github.com/SWE-bench/SWE-bench/blob/main/swebench/harness/grading.py
- SWE-bench paper: https://arxiv.org/abs/2310.06770
- SWE-bench Verified: https://openai.com/index/introducing-swe-bench-verified/
- Ante, harness held constant: https://antigma.ai/blog/2026/08/04/harness-matter
- Ante results page: https://antigma.ai/eval
- Scaffold effect (KDD 2026 workshop): https://kdd-eval-workshop.github.io/agenticai-evaluation-kdd2026/assets/papers/74_The_Scaffold_Effect_in_Codi.pdf
