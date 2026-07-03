# GizTUI Development Guide

## Quick Documentation Access
- [Documentation Hub](docs/README.md) - Complete navigation
- [Architecture Guide](docs/ARCHITECTURE.md) - Development patterns
- [Testing Guide](docs/TESTING.md) - Quality assurance
- [Theming Guide](docs/THEMING.md) - UI component theming

## Project Layout
- cmd/giztui/main.go - Entry point
- internal/services/ - Business logic (see interfaces.go)
- internal/tui/ - Presentation (app.go is central, 3k+ lines)
- internal/gmail/, internal/llm/, internal/cache/, internal/db/ - Integrations
- pkg/auth/ - OAuth2 token handling
- Runtime config: ~/.config/giztui/config.json
- OAuth creds: ~/.config/giztui/credentials.json
- First-run setup: giztui --setup

## Git Commit Guidelines
Do NOT include Claude signatures or co-authored-by lines. Keep commits clean.

## Core Architecture

### Service-First Development (MANDATORY)
- ALL business logic → internal/services/
- UI components: presentation + user input only
- NEVER: Gmail/LLM/API calls in TUI

### New Feature Steps
1. Define interface in internal/services/interfaces.go
2. Implement service in dedicated file
3. Add to App struct in internal/tui/app.go
4. Initialize in initServices()
5. Update GetServices() return values
6. Integrate UI (call service methods)

## Critical Patterns

### Error Handling (MANDATORY)
- Use app.GetErrorHandler() for ALL user feedback
- NEVER: fmt.Printf, log.Printf, direct output
- Methods: ShowError, ShowSuccess, ShowWarning, ShowInfo, ShowProgress, SetPersistentStatus

### Thread Safety (MANDATORY)
- Use accessor methods: GetCurrentView(), SetCurrentMessageID()
- NEVER access app struct fields directly
- Use proper mutex protection for new state fields

### ESC Key Handling (CRITICAL)
- NEVER use QueueUpdateDraw() in ESC handlers
- Use synchronous operations to prevent deadlocks

### Status Messages (CRITICAL)
- Use ErrorHandler for ALL status operations
- NEVER use direct status methods (setStatusPersistent, showStatusMessage)
- NEVER wrap ErrorHandler calls in QueueUpdateDraw()

### Streaming Callbacks (CRITICAL)
- NEVER use QueueUpdateDraw() in streaming callbacks
- Use direct UI updates to prevent deadlocks

### Picker State Management (MANDATORY)
- Use ActivePicker enum system (PickerNone, PickerLabels, PickerDrafts, etc.)
- NEVER use shared boolean flags like labelsVisible
- Use setActivePicker() and isLabelsPickerActive() methods

### Theming (MANDATORY)
- Use app.GetComponentColors("component") for ALL UI theming
- NEVER use deprecated theme methods or hardcoded colors
- Components: general, search, attachments, obsidian, saved_queries, slack, prompts, ai, labels, stats, links

## Command Parity (MANDATORY)
- Every keyboard shortcut MUST have equivalent :command
- Commands MUST support bulk mode and provide short aliases
- Add to executeCommand() in internal/tui/commands.go
- Examples: a → :archive/:arch, t → :trash/:tr, l → :labels/:lab

## Build & Test Commands

### Build
- make build - Build with version injection
- make deps - Install dependencies
- make dev - Development mode (build + run)
- make debug - Debug build (no optimizations)
- make run - Build and run

### Test
- make test - All tests (internal/test/pkg)
- make test-unit - Unit tests only
- make test-tui - TUI component tests
- make test-integration - Integration tests
- make test-all - Full suite with mocks/coverage
- make test-race - Race detector tests
- make test-coverage - Coverage report
- make test-mocks - Generate mocks (mockery)
- Individual: go test -v ./internal/services -run TestName

### Code Quality
- make fmt - Format code
- make lint - golangci-lint
- make vet - Verify code
- make pre-commit-check - CI-equivalent checks (use before claiming complete)

### Release
- make release-build - Cross-platform binaries
- make release - Build + archives
- make version - Version info

### Utilities
- make clean - Remove build artifacts
- make bench - Benchmarks
- make check-deps - Verify dependencies
- make update-deps - Update dependencies
- make setup-hooks - Install pre-commit hooks

## Development Workflow
1. Identify needed services → check existing → extend/create interface
2. Implement service (business logic only) in internal/services/
3. Integrate UI (presentation only) in internal/tui/
4. Use ErrorHandler, thread-safe accessors, picker enums
5. Add command parity (keyboard → :command)
6. Run: make pre-commit-check before claiming complete
7. Update :help screen and docs if needed
8. For config changes: add to DefaultConfig() + migration path

## Reference Documentation
- docs/ARCHITECTURE.md - Complete patterns
- docs/THEMING.md - Theme system
- docs/FOCUS_MANAGEMENT.md - UI focus patterns
- docs/TESTING.md - Testing framework
- docs/RELEASE_PROCEDURE.md - Release management
- docs/KEYBOARD_SHORTCUTS.md - Command system
- docs/GMAIL_SEARCH_REFERENCE.md - Gmail search operators
- internal/services/interfaces.go - All service contracts
- internal/tui/error_handler.go - Error handling
- internal/tui/keys.go - ESC key handling

## graphify
Knowledge graph at graphify-out/ with god nodes, community structure, cross-file relationships.
- graphify query "<question>" - Scoped subgraph
- graphify path "<A>" "<B>" - Relationships
- graphify explain "<concept>" - Focused concepts
- graphify update . - Update after code changes (AST-only, no API cost)
