# GitHub Actions Workflows

This directory contains CI/CD workflows for the hapctl project.

## Workflows

### CI Workflow (`ci.yml`)

Runs on every push and pull request to `main` and `develop` branches.

**Jobs:**

1. **Test**
   - Runs all unit tests with race detection
   - Generates coverage report
   - Uploads coverage as artifact

2. **Build**
   - Builds the binary
   - Verifies the binary works
   - Uploads binary as artifact

3. **Lint**
   - Runs golangci-lint for code quality checks

**Triggers:**
- Push to `main` or `develop` branches
- Pull requests to `main` or `develop` branches

### Release Workflow (`release.yml`)

Automatically creates releases when a version tag is pushed.

**Jobs:**

1. **Release**
   - Runs tests
   - Builds binaries for multiple platforms:
     - Linux (amd64, arm64)
     - macOS (amd64, arm64)
   - Generates checksums
   - Creates GitHub release with binaries

**Triggers:**
- Push of tags matching `v*` (e.g., `v1.0.0`, `v1.2.3`)

## Creating a Release

To create a new release:

```bash
# Tag the commit
git tag -a v1.0.0 -m "Release version 1.0.0"

# Push the tag
git push origin v1.0.0
```

The release workflow will automatically:
1. Run tests
2. Build binaries for all platforms
3. Create a GitHub release
4. Upload binaries and checksums

## Local Testing

Before pushing, you can run the same checks locally:

```bash
# Run tests
make test

# Build
make build

# Run linter (requires golangci-lint)
make lint
```

## Configuration

### golangci-lint

The linter configuration is in `.golangci.yml` at the project root.

Enabled linters:
- gofmt
- goimports
- govet
- errcheck
- staticcheck
- unused
- gosimple
- ineffassign
- typecheck

## Artifacts

CI runs produce artifacts that can be downloaded from the Actions tab:
- **coverage-report**: HTML coverage report
- **hapctl-binary**: Built binary for testing

## Status Badges

Add these badges to your README.md:

```markdown
[![CI](https://github.com/eliasmeireles/hapctl/actions/workflows/ci.yml/badge.svg)](https://github.com/eliasmeireles/hapctl/actions/workflows/ci.yml)
[![Release](https://github.com/eliasmeireles/hapctl/actions/workflows/release.yml/badge.svg)](https://github.com/eliasmeireles/hapctl/actions/workflows/release.yml)
```
