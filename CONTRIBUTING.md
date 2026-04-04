# Contributing to Web Crawler SDK

Thank you for your interest in contributing! This guide covers the development setup and workflow.

## Development Setup

### Prerequisites

- Go 1.25+
- Python 3.10+ (for Python SDK)
- [buf](https://buf.build/) (for Protocol Buffers)
- [golangci-lint](https://golangci-lint.run/) v1.64+

### Clone and Build

```bash
git clone https://github.com/kcenon/web_crawler.git
cd web_crawler
go build ./...
```

### Generate Protocol Buffers

```bash
cd api/proto
buf generate
```

### Run Tests

```bash
# Go tests
go test ./...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Python tests
cd python
pip install -e ".[dev]"
pytest
```

### Run Linter

```bash
golangci-lint run ./...
```

## Project Structure

```
web_crawler/
├── api/proto/          # Protocol Buffer definitions
├── cmd/crawler/        # CLI application
│   └── cmd/            # Cobra commands (crawl, run, server, init)
├── examples/           # Usage examples
├── internal/
│   └── testutil/       # Test helpers, fixtures, benchmarks
├── pkg/
│   ├── client/         # HTTP client with connection pooling
│   ├── crawler/        # Core crawler engine and Service
│   ├── extractor/      # CSS selector data extraction
│   ├── frontier/       # URL frontier with priority queue
│   ├── middleware/      # Middleware chain (retry, rate limit, robots.txt)
│   ├── observability/  # Logging and Prometheus metrics
│   ├── server/         # gRPC server
│   └── storage/        # File storage (JSON Lines, CSV)
└── python/             # Python SDK
    └── crawler/        # Client library
```

## Workflow

1. Check open issues or create a new one
2. Fork and create a feature branch: `feat/issue-<number>-<description>`
3. Make changes following existing code style
4. Ensure `go build ./...`, `golangci-lint run ./...`, and `go test ./...` pass
5. Commit with [Conventional Commits](https://www.conventionalcommits.org/): `feat(scope): description`
6. Create a PR with `Closes #<issue-number>`

## Code Style

- Follow existing patterns in the codebase
- All code, comments, and documentation in English
- No AI/Claude attribution in commits or PRs
- Use `log/slog` for structured logging
- Interfaces define contracts, implementations are package-private where possible

## Commit Messages

Format: `type(scope): description`

| Type | Usage |
|------|-------|
| `feat` | New feature |
| `fix` | Bug fix |
| `refactor` | Code restructuring |
| `docs` | Documentation |
| `test` | Tests |
| `chore` | Build, CI, dependencies |

## Architecture Notes

- **Engine** (`pkg/crawler`): Worker pool with callback-driven processing
- **Service** (`pkg/crawler`): Multi-tenant management layer for gRPC
- **Middleware** (`pkg/middleware`): Onion model with `NextFunc` composition
- **Frontier** (`pkg/frontier`): Heap-based priority queue with dedup and filtering
- **Storage** (`pkg/storage`): Plugin interface with JSON Lines and CSV implementations
