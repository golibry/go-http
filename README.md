
# go-http

A Go HTTP middleware library providing common functionalities for HTTP request/response handling, error management, access logging, and panic recovery.

Migrated from https://github.com/rsgcata/go-http

## What this library offers

- Response building helpers for JSON, text, and HTML
- Structured error handling with categories and optional logging
- Enhanced response writer that tracks HTTP status codes
- Middleware for access logging, panic recovery, request timeouts, path normalization, CSRF protection, and session management
- A simple router with middleware chaining support

## Installation

```bash
go get github.com/golibry/go-http
```

## How to learn and use it

To keep this README focused, feature-specific code examples live in the `_examples` folder. Each example is a small, self‑contained program with comments explaining how to use a single feature.

Start here:
- Browse `_examples/` for runnable, focused examples (one file per feature)
- See `http/session/README.md` for comprehensive session documentation
- Check tests under `http/**` for additional usage patterns

## Examples directory (overview)

See the `_examples` folder for:
- `response_builder.go` — Build JSON/Text/HTML and error responses
- `enhanced_error_handling.go` — Error categories and logging controls
- `timeout_middleware.go` — Request timeout middleware
- `pathnormalizer.go` — Normalize URL paths
- `access_logger.go` — Structured access logging middleware
- `recoverer_middleware.go` — Panic recovery middleware
- `csrf_middleware.go` — CSRF protection using a deliberate header
- `response_writer.go` — Track status codes with a wrapped writer
- `router_chain.go` — Chain multiple middleware
- `session_management.go` — Sessions: attributes, flash, storage backends (see also `http/session/README.md`)

## Requirements

- Go 1.24.1 or later
- Standard library for core functionality; testing uses `github.com/stretchr/testify`

## License

See the LICENSE file for details.