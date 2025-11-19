# GitHub Copilot Instructions for datacat

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

## When Making Changes

1. **Understand the architecture**: Read ARCHITECTURE.md before major changes
2. **Maintain compatibility**: Don't break Python 2.7 or existing API contracts
3. **Test thoroughly**: Run all tests locally before pushing
4. **Update documentation**: Keep docs in sync with code changes
5. **Follow conventions**: Match existing code style and patterns
6. **Keep it minimal**: Make smallest possible changes to achieve goals
