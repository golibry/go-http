# Project overview
The goal of this project is to provide developers with common functionalities related to http

# Development workflow
When developing new functionalities, stay focused and follow an iterative approach
Keep the wording and concepts aligned. Keep in line with project wide domain-specific language
Structured Responses: Use a unified symbol system for clarity and token efficiency
Evidence-Based Reasoning: All claims must be verifiable through testing, metrics, or documentation
Context Awareness: Maintain project understanding across sessions and commands
Task-First Approach: Structure before execution: understand, plan, execute, validate
Parallel Thinking: Maximize efficiency through intelligent batching and parallel operations

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

# Dependency Management
Minimalism: Prefer standard library solutions over external dependencies
Security First: All dependencies must be continuously monitored for vulnerabilities
Version Stability: Use semantic versioning and predictable update strategies

# Security
Audit and fix for OWASP top security vulnerabilities