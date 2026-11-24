# Run from repo root. Harness module lives in harness/.
HARNESS := ./harness
MUXON    := $(HARNESS)/cmd/muxon

.PHONY: help build muxon test vet fmt lint eval

help:
	@echo "Targets (from repo root):"
	@echo "  make build        - build harness/muxon binary"
	@echo "  make muxon         - go run muxon CLI (pass ARGS=...)"
	@echo "  make test         - go test ./harness/..."
	@echo "  make vet          - go vet ./harness/..."
	@echo "  make fmt          - gofmt harness/"
	@echo "  make lint         - golangci-lint run (in harness/)"
	@echo "  make eval         - OpenCode Go basket (from repo root)"

build:
	go build -o harness/muxon $(MUXON)

muxon:
	go run $(MUXON) $(ARGS)

test:
	go test ./harness/...

vet:
	go vet ./harness/...

fmt:
	gofmt -w $(HARNESS)

lint:
	cd harness && golangci-lint run

eval:
	cd harness && go run ./cmd/muxon eval basket --matrix ../evals/matrices/opencode.yaml --tasks '../evals/tasks/*.json' --trials 1 --concurrency 2 --budget-usd 5 --trust
