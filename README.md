# thee-harness

Monorepo for **thee**: an open-source, CLI-first coding agent harness (small models, verify-gated `finish`, local JSONL traces). Product overview: [docs/product.md](docs/product.md).

| Path | What it is |
|------|------------|
| [`harness/`](harness/) | Go module — `thee` CLI and all runtime code |
| [`api/`](api/) | Placeholder for the closed inference API module (not implemented here) |
| [`docs/`](docs/) | Architecture specs, research, benchmarks |

**Status:** Milestones **M0**, **A** (headless), **B** (TUI), **C** (replay and local eval report), and **D** (release stubs) are in the tree. Scripted `thee run` and `thee chat` can fix [`go-failing-test`](harness/testdata/fixtures/go-failing-test). Live model-matrix scoring, probe thresholds, and a tagged release are still open. See [implementation plan](docs/harness-architecture/implementation-plan.md). The module path is still `github.com/PLACEHOLDER/thee/harness`.

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

**Task shortcuts** (optional, works on Windows, macOS, and Linux; install from [taskfile.dev](https://taskfile.dev/installation/)):

```bash
task                # list tasks
task build          # → harness/thee (harness/thee.exe on Windows)
task thee -- version
task test
```

**Make shortcuts** (optional, Unix only):

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

## Install

Release archives are built with [`.goreleaser.yml`](.goreleaser.yml) (`linux` and `darwin`, `amd64` and `arm64`). This repo does not publish tags yet. Until then:

```bash
go build -o thee ./harness/cmd/thee
```

## Commands (milestone D)

| Command | Description |
|---------|-------------|
| `thee` / `thee chat` | Interactive TUI |
| `thee run` | Headless agent session |
| `thee resume [session-id]` | Continue the newest session, or a named one |
| `thee doctor` | Keys, providers, `rg` on PATH (never prints secrets) |
| `thee cost` | Token and USD totals from the latest session trace |
| `thee trust list\|add\|rm` | Workspace trust store under `THEE_HOME` |
| `thee models list\|show\|probe` | Merged manifest; probe writes a catalog snippet |
| `thee auth login <provider>` | Keyring entry; catalog records the provider id only |
| `thee trace record\|replay\|export` | Scripted replay file, load it, or redact a trace |
| `thee eval run\|report` | Local basket row and aggregate report |
| `thee gc` | Drop session dirs older than 14 days and `refs/thee/sessions/*` |
| `thee completion bash\|zsh\|fish` | Shell completion |
| `thee serve` | Exits 3; IDE protocol is not in this build |
| `thee version` | Print build version |

Eval task ids: `fix-test-001`, `feature-add-001`, `lint-two-file-001`, `dep-bump-001`, `rename-001`.

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
