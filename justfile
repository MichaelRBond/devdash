# Default: list available recipes
default:
    @just --list

# ─── Development ───────────────────────────────────────────

# Run the dashboard
run *ARGS:
    go run . {{ARGS}}

# Run with live reload (requires air: go install github.com/air-verse/air@latest, or use `go run`)
dev:
    go run github.com/air-verse/air@latest \
        --build.cmd "go build -o tmp/devdash ." \
        --build.bin "tmp/devdash"

# ─── Build ─────────────────────────────────────────────────

# Build the binary
build:
    go build -ldflags "-s -w -X main.version=$(git describe --tags --always --dirty)" -o bin/devdash .

# Build for all target platforms
build-all:
    GOOS=linux  GOARCH=amd64 go build -ldflags "-s -w" -o bin/devdash-linux-amd64 .
    GOOS=linux  GOARCH=arm64 go build -ldflags "-s -w" -o bin/devdash-linux-arm64 .
    GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o bin/devdash-darwin-amd64 .
    GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o bin/devdash-darwin-arm64 .

# Build and install to ~/bin
install: build
    mkdir -p ~/bin
    cp bin/devdash ~/bin/devdash

# Clean build artifacts
clean:
    rm -rf bin/ tmp/

# ─── Quality ───────────────────────────────────────────────

# Format all Go files (built-in)
fmt:
    gofmt -s -w .

# Check formatting without writing (CI-friendly)
fmt-check:
    @test -z "$(gofmt -l .)" || (echo "Files need formatting:" && gofmt -l . && exit 1)

# Vet — catch common mistakes (built-in)
vet:
    go vet ./...

# Lint — uses golangci-lint via go run (no global install needed)
lint:
    go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run ./...

# Run all quality checks
check: fmt-check vet lint

# ─── Testing ───────────────────────────────────────────────

# Run all tests
test:
    go test ./...

# Run tests with verbose output
test-v:
    go test -v ./...

# Run tests with coverage
test-cover:
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out

# Open coverage report in browser
test-cover-html: test-cover
    go tool cover -html=coverage.out

# Run tests with race detector
test-race:
    go test -race ./...

# ─── Dependencies ──────────────────────────────────────────

# Tidy module dependencies
tidy:
    go mod tidy

# Show outdated dependencies
outdated:
    go list -u -m all

# ─── Release ───────────────────────────────────────────────

# Full pre-commit check
pre-commit: fmt check test

# Tag a release (usage: just release v0.1.0)
release VERSION:
    git tag -a {{VERSION}} -m "Release {{VERSION}}"
    git push origin {{VERSION}}
