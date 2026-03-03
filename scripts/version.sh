#!/bin/bash
set -e

VERSION=$1

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo ""
    echo "Examples:"
    echo "  $0 v0.1.0        # Create release version"
    echo "  $0 v0.1.0-rc1    # Create release candidate"
    echo "  $0 v0.1.0-beta1  # Create beta version"
    echo ""
    exit 1
fi

# Validate version format (vX.Y.Z or vX.Y.Z-suffix)
if ! echo "$VERSION" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?$'; then
    echo "Error: Invalid version format. Expected: vX.Y.Z or vX.Y.Z-suffix"
    echo "Examples: v0.1.0, v1.2.3-rc1, v2.0.0-beta1"
    exit 1
fi

# Check if tag already exists
if git rev-parse "$VERSION" >/dev/null 2>&1; then
    echo "Error: Tag $VERSION already exists"
    exit 1
fi

# Check if working directory is clean
if ! git diff-index --quiet HEAD --; then
    echo "Error: Working directory has uncommitted changes"
    echo "Please commit or stash your changes first"
    exit 1
fi

# Determine if this is a pre-release
PRERELEASE=""
if echo "$VERSION" | grep -qE '-(rc|beta|alpha)'; then
    PRERELEASE=" (pre-release)"
fi

echo "Creating version: $VERSION$PRERELEASE"
echo ""

# Get current branch
BRANCH=$(git branch --show-current)
echo "Current branch: $BRANCH"

# Get commit count since last tag
COMMITS=$(git rev-list --count $(git describe --tags --abbrev=0 2>/dev/null || echo HEAD)..HEAD 2>/dev/null || echo "0")
if [ "$COMMITS" != "0" ]; then
    echo "Commits since last tag: $COMMITS"
fi

echo ""
read -p "Create and push tag $VERSION? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cancelled"
    exit 1
fi

# Create annotated tag
git tag -a "$VERSION" -m "Release $VERSION"

echo ""
echo "✅ Tag $VERSION created locally"
echo ""
echo "To push the tag and trigger release:"
echo "  git push origin $VERSION"
echo ""
echo "Or run: make version-push VERSION=$VERSION"
