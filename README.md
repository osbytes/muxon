# thee-harness

Monorepo for **thee**: an open-source, CLI-first coding agent harness (small models, verify-gated `finish`, local JSONL traces). Product overview: [docs/product.md](docs/product.md).

| Path | What it is |
|------|------------|
| [`harness/`](harness/) | Go module — `thee` CLI and all runtime code |
| [`api/`](api/) | Placeholder for the closed inference API module (not implemented here) |
| [`docs/`](docs/) | Architecture specs, research, benchmarks |

**Status:** **M0** and **milestone A** (headless CLI) are implemented — scripted `thee run` fixes the [`go-failing-test`](harness/testdata/fixtures/go-failing-test) fixture; BYOK worker binding, `doctor`, `cost`, `trust`, and `eval run` are wired. TUI and probe/eval matrix are Part 8–9; see [implementation plan](docs/harness-architecture/implementation-plan.md).

---

## Prerequisites

- **Go 1.24+** ([install](https://go.dev/doc/install))
- **Git**
- For upcoming tools: **ripgrep** (`rg`) — CI installs it; optional locally until Part 2

---

## Get started

### Clone and build

```bash
git clone git@github.com:adavila0703/thee-harness.git
cd thee-harness
```

The repo uses a [Go workspace](go.work); harness code lives under `harness/`. You can stay at the **repo root** for most commands.

**Make shortcuts** (optional):

```bash
make help
make build          # → harness/thee
make thee ARGS=version
make test
```

**Go directly** (same as above, no Make):

```bash
go run ./harness/cmd/thee version
go build -o harness/thee ./harness/cmd/thee
go test ./harness/...
```

Equivalent from inside `harness/`:

```bash
cd harness
go build -o thee ./cmd/thee
go test ./...
```

### Environment

| Variable | Default | Purpose |
|----------|---------|---------|
| `THEE_HOME` | `~/.thee` | Config, catalog, trust store (tests should set this to a temp dir) |
| `THEE_SCRIPTED_PROVIDER_FILE` | — | Path to a JSON file of canned model responses (CI / demos; overrides BYOK when set) |
| `THEE_TRUST` | — | Set to `1` to trust the workspace (same as `--trust`) |
| `DEEPINFRA_API_KEY` (etc.) | — | Provider keys per embedded `models.yaml` / `~/.thee/catalog.json` |

---

## Commands (milestone A)

Headless commands from the [CLI spec](docs/harness-architecture/cli.md) (interactive `thee chat` is Part 8).

| Command | Description |
|---------|-------------|
| `thee version` | Print build version (`dev` unless set at link time) |
| `thee run` | Headless agent session |
| `thee doctor` | Keys, providers, `rg` on PATH (never prints secrets) |
| `thee cost` | Token and USD totals from the latest session trace |
| `thee trust list\|add\|rm` | Workspace trust store under `THEE_HOME` |
| `thee eval run <task.json>` | Self-graded eval task → `.thee/eval/results.jsonl` |

Global flags: `-C` / `--directory`, `--json`, `-q` / `--quiet`, `-v` / `--verbose`.

### `thee run`

```text
thee run -p "<goal>" [--trust] [--verify-cmd "..."] [--model <id>] [--max-tool-rounds N]
         [--cost-budget-usd X] [--permission edit|full|read-only] [--yolo] [--no-plan]
         [--eval <task.json>]
```

| Flag | Description |
|------|-------------|
| `-p`, `--prompt` | Goal text (required) |
| `--trust` | Honor project verify command in the workspace |
| `--verify-cmd` | Override verify argv (whitespace-separated) |
| `--model` | Force worker model id from manifest |
| `--max-tool-rounds`, `--cost-budget-usd` | Per-run caps |
| `--permission`, `--yolo` | Tier / auto-approve within user `max_permission` |
| `--no-plan` | Skip planning (v1 records only; no planning state) |
| `--eval` | Append a row to `.thee/eval/results.jsonl` after the run |

**Exit codes:** `0` done · `1` failed (verify/budget/caps) · `2` approval required · `3` setup · `130` cancelled (SIGINT).

**Provider:** Without `THEE_SCRIPTED_PROVIDER_FILE`, `thee run` uses the **worker** role from merged `models.yaml` + `~/.thee/catalog.json` (BYOK). Set the provider’s `api_key_env` in your environment.

### Scripted M0 demo (fix the sample fixture)

1. Copy the fixture somewhere safe (the harness mutates the tree):

   ```bash
   cp -R harness/testdata/fixtures/go-failing-test /tmp/go-failing-test
   ```

2. Write the scripted responses (same sequence as [`director.M0ScriptedResponses`](harness/internal/director/director.go)) — or export them once from a small Go snippet / test helper.

3. Run:

   ```bash
   export THEE_SCRIPTED_PROVIDER_FILE=/path/to/script.json
   export THEE_HOME="${TMPDIR:-/tmp}/thee-home"
   cd harness
   go run ./cmd/thee run -p "fix the failing test" --trust --directory /tmp/go-failing-test
   ```

4. Inspect the trace (session id is timestamped; `thee cost` reads the latest):

   ```text
   /tmp/go-failing-test/.thee/sessions/<session-id>/trace.jsonl
   ```

   Expect `verify.run` (passed) and `session.end` with `outcome` `done`.

---

## Development

From the repo root:

```bash
make fmt test vet    # or: gofmt -w harness/ && go test ./harness/... && go vet ./harness/...
make lint            # golangci-lint in harness/
```

Lint (optional, matches CI):

```bash
cd harness
golangci-lint run
```

GitHub Actions runs test, vet, lint, and build on **linux** and **macOS** for changes under `harness/`.

---

## Documentation

| Topic | Link |
|-------|------|
| Architecture index | [docs/harness-architecture/README.md](docs/harness-architecture/README.md) |
| Implementation plan & milestones | [docs/harness-architecture/implementation-plan.md](docs/harness-architecture/implementation-plan.md) |
| CLI & TUI (target) | [docs/harness-architecture/cli.md](docs/harness-architecture/cli.md) |
| Closed API (separate module) | [docs/api/README.md](docs/api/README.md) |

---

## Module path

`harness/go.mod` currently uses `github.com/PLACEHOLDER/thee/harness`. Replace `PLACEHOLDER` with your GitHub org or username before publishing the module or cutting releases.

---

## License

License file not added yet; treat as private development until you declare one.
