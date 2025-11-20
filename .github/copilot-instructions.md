# GitHub Copilot Instructions for datacat

> **Quick Reference**: Start with [Working Philosophy](#working-philosophy) and [Development Workflow](#development-workflow) for key principles.

## Table of Contents
- [Working Philosophy](#working-philosophy)
- [Project Overview](#project-overview)
- [Repository Structure](#repository-structure)
- [Architecture Principles](#architecture-principles)
- [Code Standards](#code-standards)
- [Testing Requirements](#testing-requirements)
- [Development Workflow](#development-workflow)
- [Common Pitfalls to Avoid](#common-pitfalls-to-avoid)
- [File Organization](#file-organization)
- [Tool Usage Patterns](#tool-usage-patterns)
- [Key Features and Patterns](#key-features-and-patterns)
- [API Design Guidelines](#api-design-guidelines)
- [Build and Development](#build-and-development)
- [Common Development Tasks](#common-development-tasks)
- [Important Considerations](#important-considerations)
- [CI/CD Pipeline](#cicd-pipeline)
- [Documentation](#documentation)
- [Common Patterns](#common-patterns)
- [Security Considerations](#security-considerations)
- [Performance Expectations](#performance-expectations)

## Working Philosophy

**Make the smallest possible changes to achieve your goals.** This is critical:
- Only modify files that are directly related to your task
- Preserve existing functionality unless explicitly changing it
- Don't refactor unrelated code
- Don't fix unrelated bugs or test failures unless they block your work
- Use existing libraries and patterns - don't introduce new dependencies unless necessary

## Project Overview

datacat is a complete data logging system with REST API service, client libraries, and web UI for tracking application sessions, state, events, and metrics. This is a Go and Python monorepo with multiple independent programs and libraries.

## Repository Structure

```
datacat/
├── cmd/                      # Executable programs (Go)
│   ├── datacat-server/      # Main REST API service with BadgerDB
│   ├── datacat-daemon/      # Local batching daemon (subprocess)
│   └── datacat-web/         # Web UI dashboard (htmx + Chart.js)
├── client/                  # Go client library
├── python/                  # Python client library (2.7+ and 3.x compatible)
├── examples/                # Example applications (Python and Go)
├── tests/                   # Integration tests (pytest)
└── docs/                    # Documentation site (Jekyll/just-the-docs)
```

## Architecture Principles

### Daemon-Based Architecture (Recommended Pattern)
- Applications use daemon subprocess for intelligent batching
- Daemon provides 10-100x network reduction through smart filtering
- Daemon monitors parent process for crash/hang detection
- All state updates use deep merge to preserve nested data

### Direct Server Mode (Simple Pattern)
- Direct HTTP calls to server without batching
- Used for simple use cases or when daemon overhead not needed

## Code Standards

### Go Code
- **Build**: All programs in `cmd/` must compile successfully
- **Testing**: Maintain at least 80% code coverage
- **Formatting**: Use `go fmt` for all Go code
- **Conventions**: 
  - Add comments for all exported functions
  - Use table-driven tests
  - Handle errors explicitly, never ignore
  - Use mutex for thread-safe operations

### Python Code
- **Compatibility**: Must work with Python 2.7+ and 3.x
- **Formatting**: Use Black formatter (enforced in CI)
- **Type Checking**: Pass mypy type checking (`mypy python/ --ignore-missing-imports`)
- **Testing**: Maintain at least 80% code coverage
- **Conventions**:
  - Add type hints where possible
  - Follow PEP 8 conventions
  - Use docstrings for public functions/classes

## Testing Requirements

### Before Submitting Code
```bash
# Format Python code
black python/ examples/ tests/

# Type check Python code
mypy python/ --ignore-missing-imports

# Test Go code with coverage
go test -v -coverprofile=coverage.out ./...

# Build all Go programs
cd cmd/datacat-server && go build
cd ../datacat-web && go build
cd ../datacat-daemon && go build

# Test Python code with coverage
pytest tests/ -v --cov=python --cov-report=term
```

### Coverage Requirements
- Go code: Minimum 80% coverage
- Python code: Minimum 80% coverage
- Integration tests required for new API endpoints
- Unit tests required for client library changes

## Key Features and Patterns

### Deep Merge State Updates
- Never overwrite entire state objects
- Preserve nested fields not included in updates
- Example: Updating `window_state.open` preserves `window_state.active`

### Session Lifecycle
1. Create session (POST /api/sessions)
2. Update state, log events/metrics during session
3. Send heartbeats to detect hangs
4. End session explicitly (POST /api/sessions/{id}/end)

### Daemon Features
- **Batching**: Collects data for 5 seconds before sending
- **Smart Filtering**: Only sends state changes that actually changed
- **Heartbeat Monitoring**: Detects hung applications (60s timeout)
- **Parent Process Monitoring**: Detects crashes/abnormal exits
- **Auto-retry**: Queues failed requests with retry logic

## API Design Guidelines

### REST Endpoints
- Use consistent patterns: `/api/{resource}/{id}/{action}`
- Accept and return JSON
- Use appropriate HTTP status codes
- Include error messages in response body

### Client Library Methods
- Go client: Synchronous methods returning `(result, error)`
- Python client: Methods that raise exceptions on error
- Both clients: Support daemon and direct modes
- Consistent method names across languages

## Build and Development

### Starting the Stack
```bash
# Start API server (required)
cd cmd/datacat-server && go run main.go

# Start Web UI (optional)
cd cmd/datacat-web && go run main.go

# Daemon is auto-started by client libraries when use_daemon=True
```

### Common Development Tasks
- **Add API endpoint**: Update server, both clients, tests, and examples
- **Add client feature**: Update client, add tests, update README, add example
- **Update documentation**: Edit files in `docs/`, auto-deployed on merge

## Important Considerations

### Python 2.7 Compatibility
- Avoid f-strings (use `.format()` or `%` formatting)
- Import from `__future__` when needed
- Test with both Python 2.7 and 3.x if possible

### BadgerDB Persistence
- Database located at `./badger-data/` by default
- Thread-safe with mutex protection
- Survives server restarts
- Clean shutdown important to avoid corruption

### Error Handling
- Always handle errors explicitly in Go
- Use try/except appropriately in Python
- Log errors with context for debugging
- Return meaningful error messages to users

## CI/CD Pipeline

### Required Checks (must pass)
1. Python formatting (Black)
2. Python type checking (mypy)
3. Go build (all programs in cmd/)
4. Go tests with coverage
5. Python tests with coverage
6. Coverage threshold (80%)

### Branch Protection
- `main` branch protected
- All checks must pass
- At least one approval required
- Coverage must meet 80% threshold

## Documentation

- API reference: `docs/api-reference.md`
- Quick start: `QUICKSTART.md`
- Architecture: `ARCHITECTURE.md`
- Contributing: `CONTRIBUTING.md`
- Component READMEs in each directory

## Common Patterns

### Creating a Session (Python with Daemon)
```python
from datacat import create_session

session = create_session("http://localhost:8080", use_daemon=True)
session.update_state({"status": "running"})
session.log_event("startup", {"version": "1.0"})
session.heartbeat()
session.end()
```

### Creating a Session (Go with Daemon)
```go
c, err := client.NewClientWithDaemon("http://localhost:8080", "8079")
if err != nil {
    log.Fatal(err)
}
defer c.Close()

sessionID, _ := c.CreateSession()
c.UpdateState(sessionID, map[string]interface{}{"status": "running"})
c.Heartbeat(sessionID)
c.EndSession(sessionID)
```

## Security Considerations

- Never commit secrets or credentials
- Validate all user input in API handlers
- Use appropriate HTTP status codes
- Sanitize log output to prevent injection
- Handle file paths securely (prevent directory traversal)

## Performance Expectations

- Daemon reduces network traffic by 10-100x
- State updates should complete in <100ms
- Server should handle 100+ concurrent sessions
- BadgerDB provides fast reads/writes
- Web UI should load sessions in <1s

## Development Workflow

### Before Making Changes
1. **Understand the codebase**: Read ARCHITECTURE.md and CONTRIBUTING.md
2. **Run existing tests**: Verify baseline state before making changes
3. **Check for failures**: Note any pre-existing test failures (not your responsibility to fix)

### Making Changes
1. **Keep changes minimal**: Only modify what's necessary for your task
2. **One change at a time**: Make incremental changes and test each one
3. **Follow existing patterns**: Match the style and structure of surrounding code
4. **Maintain compatibility**: Don't break Python 2.7 or existing API contracts
5. **Test frequently**: Run relevant tests after each change

### After Making Changes
1. **Verify your changes**: Test the specific functionality you modified
2. **Run linters and formatters**: Ensure code style compliance
3. **Check coverage**: Ensure tests cover your changes (80% minimum)
4. **Update documentation**: Keep docs in sync with code changes
5. **Review commits**: Ensure only relevant files are committed (use .gitignore)

## Common Pitfalls to Avoid

### What NOT to Do
- ❌ Don't remove or modify working code unrelated to your task
- ❌ Don't fix unrelated bugs or test failures
- ❌ Don't refactor code that isn't part of your changes
- ❌ Don't add new dependencies without careful consideration
- ❌ Don't commit temporary files, build artifacts, or dependencies
- ❌ Don't break Python 2.7 compatibility (use `.format()` not f-strings)
- ❌ Don't ignore errors in Go code (always handle explicitly)
- ❌ Don't commit without running tests first

### What TO Do
- ✅ Make surgical, precise changes to accomplish your goal
- ✅ Preserve existing functionality and behavior
- ✅ Add tests for new functionality
- ✅ Use existing libraries and utilities
- ✅ Follow established code patterns
- ✅ Run formatters (black for Python, go fmt for Go)
- ✅ Check coverage before committing (80% minimum)
- ✅ Update relevant documentation

## File Organization

### What to Commit
- Source code changes in `cmd/`, `client/`, `python/`, `examples/`
- Test changes in `tests/`, `*_test.go` files
- Documentation updates in `docs/`, `*.md` files
- Configuration changes (go.mod, requirements-dev.txt, etc.)

### What NOT to Commit (already in .gitignore)
- Compiled binaries: `datacat`, `datacat-server`, `datacat-web`, `datacat-daemon`
- Test artifacts: `*.out`, `coverage*`, `*.test`
- Database files: `badger_data/`, `datacat_db/`, `*.db`
- Python artifacts: `__pycache__/`, `*.pyc`, `.venv/`, `dist/`, `build/`
- Temporary files in `/tmp/`

If you accidentally commit unwanted files, use `git rm --cached <file>` to unstage them and update `.gitignore`.

## Tool Usage Patterns

### Running Tests
```bash
# Test only what you changed - don't run full test suite unnecessarily
go test -v ./client  # If you changed Go client
pytest tests/test_unit.py -k "test_specific_function"  # If you changed specific functionality
```

### Formatting Code
```bash
# Always format before committing
black python/ examples/ tests/  # Python formatting
go fmt ./...  # Go formatting (automatically applied by go tooling)
```

### Building Components
```bash
# Only build what you need to test
cd cmd/datacat-server && go build  # If testing server
cd cmd/datacat-web && go build     # If testing web UI
cd cmd/datacat-daemon && go build  # If testing daemon
```

### Incremental Development
1. Make a small change
2. Run relevant tests immediately
3. Fix any issues
4. Commit when tests pass
5. Repeat

This approach catches errors early and makes debugging easier.

### Debugging and Troubleshooting

**Build Failures**
```bash
# Go build errors
go build -v ./...  # Verbose output to see what's failing
go mod tidy        # Fix dependency issues

# Python import errors
pip install -e python/  # Reinstall package in development mode
```

**Test Failures**
```bash
# Run specific test with verbose output
go test -v ./client -run TestSpecificFunction
pytest tests/test_unit.py::test_specific_function -v -s

# Check test coverage to find untested code
go test -coverprofile=coverage.out ./client && go tool cover -html=coverage.out
pytest tests/ --cov=python --cov-report=html
```

**Runtime Issues**
- Check server is running: `curl http://localhost:9090/api/sessions`
- Check daemon is running: `curl http://localhost:8079/health` (if using daemon mode)
- Look for database locks: Remove `badger_data/` directory and restart if corrupted
- Check logs: Server and daemon print detailed error messages to stdout

**Type Checking Issues (Python)**
```bash
# Fix type errors
mypy python/ --ignore-missing-imports --show-error-codes

# Common fixes:
# - Add type hints: def function(param: str) -> dict:
# - Use # type: ignore comments for unavoidable issues
```
