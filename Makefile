BINARY := bin/swag
COVER_PROFILE := coverage.out
COVERAGE_FLOOR := 75

.PHONY: all build test cover cover-verify vet fmt tidy clean snapshot tools

all: vet test build

build:
	go build -trimpath -ldflags "-s -w" -o $(BINARY) ./cmd/swag

test:
	go test ./...

cover:
	go test -coverprofile=$(COVER_PROFILE) ./...
	go tool cover -func=$(COVER_PROFILE) | tail -1
	@rm -f $(COVER_PROFILE)

cover-verify:
	go test -coverprofile=$(COVER_PROFILE) ./...
	@total=$$(go tool cover -func=$(COVER_PROFILE) | awk '/^total:/ {gsub("%",""); print $$3}'); \
	echo "total coverage: $$total% (floor $(COVERAGE_FLOOR)%)"; \
	rm -f $(COVER_PROFILE); \
	if [ "$$(echo "$$total < $(COVERAGE_FLOOR)" | bc -l)" = "1" ]; then \
		echo "coverage below the $(COVERAGE_FLOOR)% floor" >&2; exit 1; \
	fi

vet:
	go vet ./...

fmt:
	gofmt -l -w .

tidy:
	go mod tidy

clean:
	rm -rf bin coverage.out dist

snapshot: clean
	goreleaser release --snapshot --clean

tools:
	go install github.com/air-verse/air@latest
	go install github.com/goreleaser/goreleaser/v2@latest
