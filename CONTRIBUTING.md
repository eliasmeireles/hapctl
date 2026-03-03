# Contributing to hapctl

Thank you for your interest in contributing to hapctl!

## Development Setup

### Prerequisites

- Go 1.21 or later
- Make
- HAProxy (for testing)

### Getting Started

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/hapctl.git
   cd hapctl
   ```

3. Install dependencies:
   ```bash
   go mod download
   ```

4. Build the project:
   ```bash
   make build
   ```

## Development Workflow

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Code Style

We follow standard Go conventions:

- All code must be formatted with `gofmt`
- Use `goimports` for import organization
- Run `golangci-lint` before submitting

```bash
# Format code
make fmt

# Run linter
make lint
```

### Writing Tests

- All new features must include unit tests
- Use `github.com/stretchr/testify` for assertions
- Test names should follow the pattern: `must_<expected_behavior>`
- Use table-driven tests when appropriate

Example:
```go
func TestFeature(t *testing.T) {
    t.Run("must handle valid input", func(t *testing.T) {
        result, err := Feature(validInput)
        require.NoError(t, err)
        require.Equal(t, expected, result)
    })
}
```

## Pull Request Process

1. Create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes and commit:
   ```bash
   git add .
   git commit -m "feat: add your feature description"
   ```

3. Push to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

4. Open a Pull Request with:
   - Clear description of changes
   - Reference to any related issues
   - Test results

### Commit Message Format

Follow conventional commits:

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `test:` - Test additions or changes
- `refactor:` - Code refactoring
- `chore:` - Maintenance tasks

## Code Review

All submissions require review. We use GitHub pull requests for this purpose.

### Review Criteria

- Code follows Go best practices
- All tests pass
- Code coverage is maintained or improved
- Documentation is updated
- No breaking changes (unless discussed)

## Continuous Integration

Our CI pipeline runs on every PR:

- **Tests**: All unit tests must pass
- **Build**: Binary must build successfully
- **Lint**: Code must pass linting checks

You can run these checks locally before pushing:

```bash
make test
make build
make lint
```

## Questions?

Feel free to open an issue for:
- Bug reports
- Feature requests
- Questions about the codebase
- Documentation improvements

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
