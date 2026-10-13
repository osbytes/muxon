# Run from repo root. Harness module lives in harness/.
HARNESS := ./harness
THEE    := $(HARNESS)/cmd/thee

.PHONY: help build thee test vet fmt lint

help:
	@echo "Targets (from repo root):"
	@echo "  make build   - build harness/thee binary"
	@echo "  make thee    - go run thee CLI (pass ARGS=...)"
	@echo "  make test    - go test ./harness/..."
	@echo "  make vet     - go vet ./harness/..."
	@echo "  make fmt     - gofmt harness/"
	@echo "  make lint    - golangci-lint run (in harness/)"

build:
	go build -o harness/thee $(THEE)

thee:
	go run $(THEE) $(ARGS)

test:
	go test ./harness/...

vet:
	go vet ./harness/...

fmt:
	gofmt -w $(HARNESS)

lint:
	cd harness && golangci-lint run
