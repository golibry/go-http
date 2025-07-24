# Project Overview
The goal of this project is to provide developers with common functionalities related to HTTP middleware and request/response handling in Go.

## Domain-Specific Context
- **HTTP Middleware**: Components that intercept and process HTTP requests/responses
- **Error Handling**: Structured error classification with HTTP status code mapping
- **Logging**: Structured logging using Go's slog package for observability
- **Request Processing**: Chain-of-responsibility pattern for request handling

# Development Workflow

## Step-by-Step Development Process
1. **Understand**: Analyze the requirement and identify affected components
2. **Plan**: Design the solution considering middleware patterns and error handling
3. **Implement**: Write code following established patterns (see the Code Patterns section)
4. **Test**: Create comprehensive tests using testify suite
5. **Validate**: Ensure integration with the existing middleware chain works correctly

## Core Principles
- **Iterative Approach**: Develop incrementally with frequent validation
- **Domain Alignment**: Use consistent HTTP/middleware terminology throughout
- **Evidence-Based**: All decisions must be testable and measurable
- **Context Awareness**: Maintain understanding of the entire middleware chain
- **Structured Execution**: Always plan before implementing
- **Maintenance costs**: Add enough code that justifies the return on investment (more lines = 
  more maintenance costs)

# Code Patterns

## HTTP Middleware Structure
When creating middleware components, prefer this pattern (options and logger to be used when 
they are really needed):

    type MiddlewareName struct {
        next    http.Handler        // or CustomHandler for error-returning handlers
        ctx     context.Context     // for structured logging context
        logger  *slog.Logger       // structured logging
        options OptionsStruct      // configuration options
    }

    func NewMiddlewareName(next http.Handler, ctx context.Context, logger *slog.Logger, options OptionsStruct) *MiddlewareName {
        return &MiddlewareName{next, ctx, logger, options}
    }

    func (m *MiddlewareName) ServeHTTP(rw http.ResponseWriter, rq *http.Request) {
        // Pre-processing logic
        
        m.next.ServeHTTP(rw, rq)
        
        // Post-processing logic
    }

## Error Handling Patterns
- Use HTTPError interface for errors that map to specific HTTP status codes
- Implement error categories for flexible error classification
- Always provide fallback error handling (stderr output when logger unavailable)
- Use structured logging with context for all errors

## Logging Patterns
- Use slog.Logger with structured attributes
- Include request context in all log entries
- Log timing information for performance monitoring
- Use consistent log message formats across middleware

# Development Principles
DRY: Abstract common functionality, remove duplication
KISS: Prefer simplicity to complexity in all design decisions
YAGNI: Implement only current requirements, avoid speculative features
Separation of Concerns: Divide program functionality into distinct sections
Loose Coupling: Minimize dependencies between components
High Cohesion: Related functionality should be grouped together logically

# Decision-Making
Systems Thinking: Consider ripple effects across the entire system architecture
Long-term Perspective: Evaluate decisions against multiple time horizons
Stakeholder Awareness: Balance technical perfection with business constraints
Risk Calibration: Distinguish between acceptable risks and unacceptable compromises
Architectural Vision: Maintain a coherent technical direction across projects
Debt Management: Balance technical debt accumulation with delivery pressure

# Error Handling
Fail Fast, Fail Explicitly: Detect and report errors immediately with meaningful context
Never Suppress Silently: All errors must be logged, handled, or escalated appropriately
Context Preservation: Maintain full error context for debugging and analysis
Recovery Strategies: Design systems with graceful degradation

# Testing
github.com/stretchr/testify library should be preferred
Use testify test suit
Follow behavioral testing patterns when building tests, for example, use naming like testItCanDoSomething
Where possible prefer data providers instead of repeating the testing boilerplate code
Testing Pyramid: Emphasize unit tests, support with integration tests
Tests as Documentation: Tests should serve as executable examples of system behavior
Comprehensive Coverage: Test all critical paths and edge cases thoroughly

## HTTP Middleware Testing Patterns
- Test middleware in isolation using mock handlers
- Verify both successful and error scenarios
- Test timing and logging behavior
- Use httptest.ResponseRecorder for response validation
- Test middleware chaining and order dependencies
- Validate HTTP status codes and response headers
- Test with various request types and edge cases

# Dependency Management
Minimalism: Prefer standard library solutions over external dependencies
Security First: All dependencies must be continuously monitored for vulnerabilities
Version Stability: Use semantic versioning and predictable update strategies

# Security
Audit and fix for OWASP top security vulnerabilities