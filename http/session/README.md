# Session Management

Session management utilities for Go HTTP applications.

## Features

- Session attributes (key-value data)
- Flash messages (auto-removed after retrieval)
- Lifecycle controls (auto-create, idle timeout, expiration)
- Optional AES-GCM encryption for sensitive data
- Garbage collection of expired sessions
- Pluggable storage (in-memory and MySQL)
- Middleware integration for automatic save/load

## Usage & Examples

This README intentionally contains no code examples. For a complete, runnable walkthrough of setup and usage, see:

- `_examples/session_management.go`

Additional patterns are demonstrated in tests under:

- `http/session/**`
- `http/session/storage/**` (MySQL covered in `mysql_test.go`)

## Key Concepts

- Manager: creates, retrieves, persists sessions and runs GC
- Storage: interface-based backends (memory, MySQL, or custom)
- Middleware: `SessionMiddleware` wires sessions into the HTTP pipeline and auto-saves
- Options: cookie settings, idle timeout, encryption key, security flags

## Configuration Overview

Common configuration areas:

- Cookie: name, domain, path, secure, httpOnly, sameSite
- Timeouts: idle timeout and absolute expiration
- Security: optional encryption key (AES-GCM)
- Storage: choose memory or MySQL storage

## Security Considerations

- Use HTTPS in production (secure cookies)
- Enable encryption for sensitive data
- Set `HttpOnly` and appropriate `SameSite` values
- Run garbage collection at reasonable intervals

## Requirements

- Go 1.24.1 or later
- No external dependencies for core functionality