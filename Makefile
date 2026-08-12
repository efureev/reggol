#!/usr/bin/make

SHELL = /bin/sh

# Keep in step with .github/workflows/test.yml: a different patch release
# analyses differently.
GOLANGCI_VERSION = v2.7.2
DC_RUN_ARGS = --rm --user "$(shell id -u):$(shell id -g)"

.PHONY : help fmt lint lint-install test race alloc bench bench-gate fuzz cover check golden screenshot clean shell
.DEFAULT_GOAL : help
.SILENT : lint test race alloc

help: ## Show this help
	@printf "\033[33m%s:\033[0m\n" 'Available commands'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[32m%-11s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## Local targets. These run natively and mirror what CI runs; the toolchain
## versions match, so there is no reason to pay for Docker unless Go is missing.

fmt: ## Format the source
	gofmt -s -w .
	go mod tidy

lint: ## Run the linter, exactly as CI does
	golangci-lint run

lint-install: ## Install the exact linter version CI uses
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)

race: ## Run tests under the race detector, exactly as CI does
	go test -race ./...

alloc: ## Run the zero-allocation gates (must not run under -race)
	go test -run 'ZeroAllocations|PoolCeiling' -v ./...

test: race alloc ## Run every test CI runs

bench: ## Run benchmarks
	go test -run '^$$' -bench=. -benchmem ./...

bench-gate: ## Fail if any benchmark allocates (the CI gate)
	./scripts/check-allocs.sh ./...

fuzz: ## Run a short fuzzing campaign
	for target in FuzzParseLevel FuzzJSONEncoder FuzzTextEncoder FuzzValueAppend; do \
		go test -run '^$$' -fuzz="^$$target$$" -fuzztime=30s . || exit 1; \
	done

golden: ## Regenerate the encoder golden files in testdata
	go test -run TestEncoderGolden -update .

screenshot: ## Regenerate the console screenshot in .assets
	go run ./scripts/gen-screenshot

cover: ## Report test coverage (build tooling in scripts/ excluded, as in CI)
	go test -covermode=atomic -coverprofile=cover.out $$(go list ./... | grep -v '/scripts/')
	go tool cover -func=cover.out | tail -1

check: fmt lint test bench-gate ## Full gate before committing

## Containerised targets, for machines without a local Go toolchain.

docker-lint: ## Run the linter in Docker
	docker compose run --rm --no-deps golint golangci-lint run

docker-test: ## Run tests in Docker
	docker compose run $(DC_RUN_ARGS) --no-deps go go test -race ./...

shell: ## Start a shell in the Go container
	docker compose run $(DC_RUN_ARGS) go sh

clean: ## Tear down containers and remove generated files
	docker compose down -v -t 1
	rm -f cover.out
