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

## Terminal-Bench and Harbor

[Terminal-Bench](https://github.com/harbor-framework/terminal-bench-2) is the set harness authors use when they want to compare agents. A task is a container, an instruction, a human-written solution, and a pytest suite. The agent gets a shell. It does not get the tests.

**Terminal-Bench 2.1**, the set Ante and others publish against, is **89 tasks × 5 trials = 445 trials**. Pin the dataset tag (`terminal-bench@2.1`). The count is a property of that tag.

[Harbor](https://www.harborframework.com/docs/agents) is the runner:

1. Build the task container from `task.toml` (timeouts and resource limits live here).
2. Run the agent until it exits or the timeout fires.
3. Run `tests/test.sh`. That script writes `/logs/verifier/ctrf.json` (per-test detail) and `/logs/verifier/reward.txt`.
4. `reward.txt` is `1` only if every pytest test passed. Otherwise `0`.

The headline is the **mean reward over trials**:

```text
score = (number of trials with reward 1) / (number of trials)
```

Five trials of one task are five rows, not one row that passes if any trial passed. That second formula is pass@5. Official tables are the mean, not pass@k.

Ante’s published 82.7% is `368 / 445` on DeepSeek V4 Flash, with a standard error of about 1.79 points ([their eval page](https://antigma.ai/eval)). The binomial standard error is `sqrt(p * (1 - p) / n)` with `n = 445`. Trials inside a task are correlated, so that interval is tight. A task-clustered bootstrap is the more honest interval. The Terminal-Bench paper reports 95% intervals from repeated trials ([arXiv:2601.11868](https://arxiv.org/html/2601.11868)).

**The timeout in `task.toml` is part of the test.** A trial that is still running after that limit is a failure even if the tests would have passed later. Public Harbor jobs can be checked: duration against the task’s `timeout` field. A community review of one Ante Harbor job found trials on `caffe-cifar-10` running for hours against a 3600 second limit. Under the official rule those trials fail, and the 82.7% headline does not survive the check. When thee publishes a number, the report includes “timeouts unchanged” and the job log.

Harbor’s own comparison of a built-in agent and an installed CLI on the same model moved the score by about a point (35.2% ± 3.1 versus 36.4% ± 2.3) from stability alone. Name the Harbor version.

## Aider’s polyglot bench

Aider is the harness that **scores itself inside its own repository**. `benchmark/benchmark.py` shells out to the `aider` CLI. The suite is 225 Exercism exercises across C++, Go, Java, JavaScript, Python, and Rust ([benchmark README](https://github.com/Aider-AI/aider/blob/main/benchmark/README.md)). An earlier published run used 133 Python exercises ([aider.chat/docs/benchmarks](https://aider.chat/docs/benchmarks.html)).

For each exercise the runner copies the stub, sends the instructions, lets the model edit, then runs the language’s test command (`pytest`, `cargo test`, `go test ./...`, and so on). The model sees test **failures**, not test source. The original writeup sent only the first 50 lines of failures.

```text
pass_rate_k = 100 * (exercises with all tests passing after attempt k) / completed_tests
```

With the usual `--tries 2`, `pass_rate_1` is the first edit and `pass_rate_2` is after one repair that includes the test output. The summary also records cost, seconds per case, and the percent of cases whose edit parsed (`percent_cases_well_formed`). That last number is the one that measures the harness rather than the coding skill: a small model can lose the task by emitting an edit the harness cannot apply.

This bench fits thee’s question better than SWE-bench does. It is an edit-format test, it is cheap enough to re-run, and the grader is a unit-test command.

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

Compare against one fat-context frontier run on the same tasks, with the same price table. That is the 3× cost claim, and it only counts when pass rate stays in the same band.

### 2. An edit bench in the Aider shape

Vendor a subset of the polyglot exercises (start with the Python slice, 133, or a 25-exercise smoke set). A runner in this repo copies the stub, invokes `thee`, runs the language test, and on failure sends one repair turn with the first 50 lines of test output. Publish `pass_rate_1`, `pass_rate_2`, `percent_cases_well_formed`, seconds per case, and cost. This is the direct test of patch-only edits on a small model, and the definition matches a bench other people already understand.

### 3. Public benches through their own graders

Do not re-grade SWE-bench or Terminal-Bench inside Go.

**Harbor.** A small Python adapter in this repo subclasses Harbor’s installed-agent base. Harbor installs the thee binary in the task container and runs it headless on the instruction. Harbor’s `test.sh` writes `reward.txt`. Thee never touches that file. The command is `harbor run -d terminal-bench@2.1 -a ./evals/harbor:TheeAgent`. A pull request runs one task. A nightly or manual job runs the 89 × 5 when there is a budget. Publish the Harbor job id, the git SHA, the model id, `n_attempts`, and a line that timeouts and memory limits were left at the dataset values.

**SWE-bench.** Headless thee writes `predictions.jsonl`. Then:

```text
python -m swebench.harness.run_evaluation \
  --dataset_name princeton-nlp/SWE-bench_Lite \
  --predictions_path predictions.jsonl \
  --run_id <new id>
```

Lite (300) is the iteration set. Verified (500) is what a public claim has to name, with the grader commit pinned. Full 2,294 is optional. Docker is required. A fresh `run_id` is required whenever the patch changes.

Traces stay on the machine. A published job lists task ids, rewards, token counts, and durations. Prompt and completion bodies stay in the local JSONL unless someone explicitly exports them.

## What a thee report contains

One block, every time:

- Dataset and version (`terminal-bench@2.1`, `SWE-bench_Lite`, or `thee-basket`).
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
- Terminal-Bench 2: https://github.com/harbor-framework/terminal-bench-2
- Terminal-Bench paper: https://arxiv.org/html/2601.11868
- Harbor agents: https://www.harborframework.com/docs/agents
- Harbor custom agents: https://docs.harborframework.com/agents/custom-agents
- Aider benchmark README: https://github.com/Aider-AI/aider/blob/main/benchmark/README.md
- Aider edit-format writeup: https://aider.chat/docs/benchmarks.html
- Ante, harness held constant: https://antigma.ai/blog/2026/08/04/harness-matter
- Ante results page: https://antigma.ai/eval
- Scaffold effect (KDD 2026 workshop): https://kdd-eval-workshop.github.io/agenticai-evaluation-kdd2026/assets/papers/74_The_Scaffold_Effect_in_Codi.pdf
