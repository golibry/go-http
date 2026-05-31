
# go-http

A Go HTTP middleware library for building robust request/response pipelines with structured errors, logging, routing, and sessions.

Migrated from https://github.com/rsgcata/go-http

## Features

- Response utilities
  - ResponseBuilder for JSON, text, and HTML
  - Enhanced ResponseWriter that tracks status codes
- Error handling
  - `HTTPError` interface and error categories
  - Optional structured logging with context
- Middleware
  - Access logging, panic recovery, timeouts, path normalization, CSRF protection, session management
- Router utilities
  - Named middleware chaining with per-route overrides
- Sessions
  - Manager, middleware integration, memory/MySQL/PostgreSQL storage, flashes, GC lifecycle

## Usage & Examples

This README intentionally avoids code examples. For in‑depth, runnable examples of every feature, see the `_examples/` directory in the repository.

- Start with `_examples/` to explore focused, one-file examples
- See `http/session/README.md` for a sessions overview

## Requirements

- Go 1.24.1 or later
- Standard library for core functionality; testing uses `github.com/stretchr/testify`

## Testing

Normal unit tests do not require Docker:

```bash
go test ./...
```

MySQL and PostgreSQL session storage integration tests use testcontainers and are behind the `integration` build tag:

```bash
go test -tags=integration ./http/session/storage
```

## License

See the LICENSE file for details.
