# swag (Subtitles With A Gopher) — Makefile
# Default goal prints the help text.

BINARY := bin/swag
GO ?= go
GORELEASER ?= goreleaser
COVERAGE_FLOOR := 75

.DEFAULT_GOAL := help

.PHONY: help build install run test cover cover-html cover-verify coverage-svg \
        samples vet fmt tidy watch release-test tag snapshot clean tools

help: ## Show this help
	@printf "swag — Subtitles With A Gopher\n\n"
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-14s\033[0m %s\n", $$1, $$2}'

build: ## Build the swag binary into bin/
	$(GO) build -trimpath -ldflags "-s -w" -o $(BINARY) ./cmd/swag

install: ## Install the CLI into $GOPATH/bin (go install)
	$(GO) install -trimpath -ldflags "-s -w" ./cmd/swag

run: ## Run the CLI (extra args after --)
	$(GO) run ./cmd/swag -- $(filter-out $@,$(MAKECMDGOALS))

test: ## Run all tests with coverage summary
	$(GO) test -cover ./...cover: ## Run tests and print the per-function coverage breakdown
	$(GO) test -coverpkg=./... -coverprofile=coverage.out ./... -count=1
	@$(GO) tool cover -func=coverage.out

coverage-svg: ## Regenerate the coverage badge from the test run
	$(GO) test -coverpkg=./... -coverprofile=coverage.out ./... -count=1
	@go-test-coverage -p coverage.out -b coverage.svg 2>/dev/null || \
		echo "install go-test-coverage to regenerate the badge (make tools)"

samples: ## Pull the upstream YTSubConverter samples for the end-to-end tests
	sh scripts/fetch-samples.sh

cover-html: ## Run tests and open the HTML coverage report in a browser
	$(GO) test -coverpkg=./... -coverprofile=coverage.out ./... -count=1
	$(GO) tool cover -html=coverage.out

cover-verify: ## Fail when module coverage sits below the 75% floor
	$(GO) test -coverpkg=./... -coverprofile=coverage.out ./... -count=1
	@total=$$($(GO) tool cover -func=coverage.out | awk '/^total:/ {gsub("%",""); print $$3}'); \
	echo "total coverage: $$total% (floor $(COVERAGE_FLOOR)%)"; \
	rm -f coverage.out; \
	if [ "$$(echo "$$total < $(COVERAGE_FLOOR)" | bc -l)" = "1" ]; then \
		echo "coverage below the $(COVERAGE_FLOOR)% floor" >&2; exit 1; \
	fi

vet: ## Run go vet over all packages
	$(GO) vet ./...

fmt: ## Format all Go source with gofmt
	gofmt -w .

tidy: ## Tidy the Go module files
	$(GO) mod tidy

watch: ## Hot-reload cmd/swag on save (needs air)
	air

release-test: ## Dry-run the release: build every target into dist/ (no upload)
	$(MAKE) build
	$(MAKE) snapshot

snapshot: ## Test GoReleaser locally in snapshot mode
	$(GORELEASER) release --snapshot --clean

tag: ## Bump the version, tag, and push (the tag triggers the release)
	@CURRENT=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	MAJOR=$$(echo "$$CURRENT" | sed 's/^v//' | cut -d. -f1); \
	MINOR=$$(echo "$$CURRENT" | sed 's/^v//' | cut -d. -f2); \
	SUGGEST="v$$MAJOR.$$(($$MINOR + 1)).0"; \
	read -p "Enter version [$$SUGGEST]: " TAG; \
	TAG=$${TAG:-$$SUGGEST}; \
	if git rev-parse "$$TAG" >/dev/null 2>&1; then \
		echo "Tag $$TAG already exists, pushing..."; \
	else \
		git tag -a "$$TAG" -m "Release $$TAG" && echo "Created tag $$TAG."; \
	fi; \
	git push origin "$$TAG"

clean: ## Remove build artefacts
	$(GO) clean -cache -test-cache 2>/dev/null || true
	rm -rf bin dist coverage.out build-errors.log

tools: ## Install the development tools (air, goreleaser, go-test-coverage)
	go install github.com/air-verse/air@latest
	go install github.com/goreleaser/goreleaser/v2@latest
	go install github.com/vladopajic/go-test-coverage/v2@latest
