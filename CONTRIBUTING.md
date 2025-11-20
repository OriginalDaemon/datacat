# Contributing to datacat

We welcome contributions to datacat! This document provides guidelines for contributing to the project.

## Repository Structure

datacat is organized as a monorepo containing multiple programs and libraries:

```
datacat/
├── cmd/                      # Executable programs
│   ├── datacat-server/      # Main REST API service
│   └── datacat-web/         # Web UI dashboard
├── client/                  # Go client library
├── python/                  # Python client library
├── examples/                # Example applications
├── tests/                   # Integration tests
└── docs/                    # Documentation site
```

## Code Quality Requirements

All pull requests must pass the following checks before merging:

### Python Code
- ✅ **Black formatting** - Code must be formatted with Black
- ✅ **mypy type checking** - No type errors allowed
- ✅ **Tests** - All tests must pass
- ✅ **Coverage** - Maintain at least 80% code coverage

### Go Code
- ✅ **Build** - All programs must compile successfully
- ✅ **Tests** - All tests must pass
- ✅ **Coverage** - Maintain at least 80% code coverage
- ✅ **go fmt** - Code should be formatted with go fmt

## Development Workflow

### 1. Fork and Clone

```bash
git clone https://github.com/YOUR_USERNAME/datacat.git
cd datacat
```

### 2. Setup Development Environment

**On Windows (PowerShell):**
```powershell
.\scripts\setup.ps1
```

This will create a Python virtual environment at `.venv/` and install all dependencies.

**On Linux/macOS:**
```bash
# Install Python dependencies
pip install -r requirements-dev.txt
pip install -e python/

# Install Go dependencies
go mod download
```

### 3. Create a Branch

```bash
git checkout -b feature/your-feature-name
```

### 4. Make Changes

Follow the structure:
- **Server changes**: `cmd/datacat-server/`
- **Web UI changes**: `cmd/datacat-web/`
- **Go client changes**: `client/`
- **Python client changes**: `python/`
- **Documentation**: `docs/`

### 5. Run Tests Locally

**On Windows (PowerShell):**
```powershell
# Format Python code
.\scripts\format.ps1

# Check code quality (Black + mypy)
.\scripts\lint.ps1

# Run all tests (Go + Python)
.\scripts\test-all.ps1

# Run tests with coverage
.\scripts\test-all.ps1 -Coverage
```

**On Linux/macOS:**
```bash
# Format Python code
black python/ examples/ tests/

# Type check Python code
mypy python/ --ignore-missing-imports

# Test Go client library
go test -v -coverprofile=coverage.out ./client

# Build all Go programs
cd cmd/datacat-server && go build
cd ../datacat-web && go build

# Run integration tests
pip install -r requirements-dev.txt
pip install -e python/
cd cmd/datacat-server && go build -o ../../datacat && cd ../..
pytest tests/ -v --cov=python
```

### 6. Commit and Push

```bash
git add .
git commit -m "Brief description of changes"
git push origin feature/your-feature-name
```

### 7. Create Pull Request

- Go to GitHub and create a pull request
- Fill out the PR template
- Ensure all CI checks pass

## Building Components

### datacat-server (Main API Service)

```bash
cd cmd/datacat-server
go build -o datacat-server
./datacat-server
```

### datacat-web (Web Dashboard)

```bash
cd cmd/datacat-web
go build -o datacat-web
./datacat-web
```

### Go Client Library

```bash
go test -v ./client
```

### Python Client Library

```bash
cd python
pip install -e .
```

## Testing

### Running All Tests

From the root directory:

```bash
# Go tests
go test -v ./...

# Python tests (requires datacat-server running)
pytest tests/ -v
```

### Coverage Reports

```bash
# Go coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Python coverage
pytest tests/ -v --cov=python --cov-report=html
```

## Documentation

Documentation is built with Jekyll using the just-the-docs theme.

To update documentation:
1. Edit files in `docs/`
2. Documentation builds automatically on PRs
3. Deployed to GitHub Pages on merge to `main`

Local preview:
```bash
cd docs
bundle install
bundle exec jekyll serve
```

## Code Style

### Go Code
- Use `go fmt` for formatting
- Follow standard Go conventions
- Add comments for exported functions
- Write table-driven tests

### Python Code
- Use Black for formatting (automatically enforced)
- Add type hints where possible
- Follow PEP 8 conventions
- Maintain Python 2.7+ compatibility for client library

## Adding New Features

### New API Endpoint

1. Add handler in `cmd/datacat-server/main.go`
2. Update API documentation in `docs/api-reference.md`
3. Add client methods to `client/client.go` and `python/datacat.py`
4. Write tests in `client/client_test.go` and `tests/test_integration.py`
5. Add example usage to `examples/`

### New Client Feature

1. Implement in respective client (`client/` or `python/`)
2. Write comprehensive tests
3. Update client README
4. Add example demonstrating the feature

## Getting Help

- **Issues**: https://github.com/OriginalDaemon/datacat/issues
- **Discussions**: https://github.com/OriginalDaemon/datacat/discussions
- **Documentation**: https://OriginalDaemon.github.io/datacat/

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
