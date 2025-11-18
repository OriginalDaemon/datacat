# Branch Protection Rules

This document outlines the required branch protection rules for the `main` branch to ensure code quality and stability.

## Required Status Checks

The following GitHub Actions checks must pass before a PR can be merged to `main`:

### 1. Python Linting and Formatting
- **Job Name**: `lint-and-format`
- **Requirements**:
  - Code must pass Black formatting checks (`black --check`)
  - Code must pass mypy type checking
- **Purpose**: Ensures consistent code style and type safety

### 2. Go Tests with Coverage
- **Job Name**: `test-go`
- **Requirements**:
  - Go code must build successfully
  - All Go tests must pass
  - Coverage must be collected and reported
- **Purpose**: Validates Go code functionality and test coverage

### 3. Python Integration Tests with Coverage
- **Job Name**: `test-python`
- **Requirements**:
  - All integration tests must pass
  - Coverage must meet minimum threshold (80%)
- **Purpose**: Validates end-to-end functionality and Python client

### 4. Data Persistence Tests
- **Job Name**: `test-persistence`
- **Requirements**:
  - Data persistence tests must pass
  - Verifies data survives service restarts
- **Purpose**: Ensures data durability and BadgerDB integration

## Coverage Requirements

### Minimum Coverage Thresholds
- **Overall Project**: 80% minimum
- **Per Patch**: 80% minimum
- **Threshold**: ±2% variation allowed

### Coverage Enforcement
- Codecov automatically comments on PRs with coverage reports
- PRs with coverage below 80% will be blocked from merging
- Coverage is tracked separately for Go and Python code

## Setting Up Branch Protection

To enable these requirements on GitHub:

1. Go to **Repository Settings** → **Branches**
2. Add branch protection rule for `main`
3. Enable the following settings:

### Required Settings
- ✅ **Require a pull request before merging**
  - Require approvals: 1 (recommended)
  - Dismiss stale pull request approvals when new commits are pushed
  
- ✅ **Require status checks to pass before merging**
  - Require branches to be up to date before merging
  - Required status checks:
    - `lint-and-format`
    - `test-go`
    - `test-python`
    - `test-persistence`
    - `codecov/project` (80% coverage)
    - `codecov/patch` (80% coverage)

- ✅ **Require conversation resolution before merging**

- ✅ **Do not allow bypassing the above settings**

### Optional but Recommended
- ✅ Require linear history
- ✅ Include administrators (enforce rules for admins too)

## Workflow

### For Contributors

1. **Create a feature branch** from `main`
2. **Make your changes** with proper tests
3. **Run locally before pushing**:
   ```bash
   # Format Python code
   black python/ examples/ tests/
   
   # Type check Python code
   mypy python/ --ignore-missing-imports
   
   # Test Go code
   go test -v -coverprofile=coverage.out ./...
   
   # Test Python code
   pytest tests/ -v --cov=python --cov-report=term
   ```
4. **Push your branch** and create a PR
5. **Wait for CI checks** to complete
6. **Address any failures**:
   - Fix formatting issues if Black check fails
   - Fix type errors if mypy check fails
   - Add tests if coverage is below 80%
   - Fix failing tests
7. **Request review** once all checks pass
8. **Merge** after approval and passing checks

### For Maintainers

1. **Review code quality** and test coverage
2. **Ensure all status checks pass**
3. **Verify coverage reports** from Codecov
4. **Approve and merge** only if all requirements are met

## Troubleshooting

### Black Formatting Fails
```bash
# Auto-fix formatting issues
black python/ examples/ tests/
```

### Mypy Type Checking Fails
```bash
# Run mypy to see errors
mypy python/ --ignore-missing-imports

# Add type hints or use type: ignore comments for false positives
```

### Coverage Below 80%
```bash
# Run with coverage report to see what's missing
pytest tests/ -v --cov=python --cov-report=html

# Open htmlcov/index.html to see detailed coverage
# Add tests for uncovered code
```

### Tests Failing
```bash
# Run specific test to debug
pytest tests/test_integration.py::TestClassName::test_method -v

# Check logs for errors
# Fix the code or update tests as needed
```

## Exceptions

In rare cases where requirements cannot be met (e.g., external dependencies), maintainers with admin access can override protections. However, this should be documented and justified in the PR.

## Benefits

These requirements ensure:
- **Code Quality**: Consistent formatting and type safety
- **Test Coverage**: Comprehensive test suite with 80%+ coverage
- **Stability**: All tests pass before merging
- **Documentation**: Changes are well-tested and documented
- **Maintainability**: Future changes are easier with good test coverage
