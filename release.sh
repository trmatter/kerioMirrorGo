#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored messages
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if git is installed
if ! command -v git &> /dev/null; then
    print_error "Git is not installed. Please install git first."
    exit 1
fi

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    print_error "Not a git repository. Please run this script from the project root."
    exit 1
fi

# Check for uncommitted changes
if ! git diff-index --quiet HEAD --; then
    print_error "You have uncommitted changes. Please commit or stash them first."
    git status --short
    exit 1
fi

# Get the latest tag
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
print_info "Latest tag: $LATEST_TAG"

# Remove 'v' prefix if present
LATEST_VERSION=${LATEST_TAG#v}

# Split version into components
IFS='.' read -r -a VERSION_PARTS <<< "$LATEST_VERSION"
MAJOR=${VERSION_PARTS[0]:-0}
MINOR=${VERSION_PARTS[1]:-0}
PATCH=${VERSION_PARTS[2]:-0}

# Determine the new version based on input
if [ $# -eq 0 ]; then
    print_error "Usage: $0 <major|minor|patch|x.y.z>"
    echo ""
    echo "Examples:"
    echo "  $0 patch   # Increment patch version (e.g., 1.7.0 → 1.7.1)"
    echo "  $0 minor   # Increment minor version (e.g., 1.7.0 → 1.8.0)"
    echo "  $0 major   # Increment major version (e.g., 1.7.0 → 2.0.0)"
    echo "  $0 1.8.5   # Set specific version"
    exit 1
fi

case "$1" in
    major)
        NEW_MAJOR=$((MAJOR + 1))
        NEW_MINOR=0
        NEW_PATCH=0
        ;;
    minor)
        NEW_MAJOR=$MAJOR
        NEW_MINOR=$((MINOR + 1))
        NEW_PATCH=0
        ;;
    patch)
        NEW_MAJOR=$MAJOR
        NEW_MINOR=$MINOR
        NEW_PATCH=$((PATCH + 1))
        ;;
    *)
        # Check if argument is a valid version number (x.y.z)
        if [[ $1 =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
            IFS='.' read -r -a NEW_VERSION_PARTS <<< "$1"
            NEW_MAJOR=${NEW_VERSION_PARTS[0]}
            NEW_MINOR=${NEW_VERSION_PARTS[1]}
            NEW_PATCH=${NEW_VERSION_PARTS[2]}
        else
            print_error "Invalid argument: $1"
            print_error "Use 'major', 'minor', 'patch', or a specific version (e.g., 1.8.5)"
            exit 1
        fi
        ;;
esac

NEW_VERSION="$NEW_MAJOR.$NEW_MINOR.$NEW_PATCH"
NEW_TAG="v$NEW_VERSION"

print_info "New version: $NEW_TAG"

# Confirm before proceeding
read -p "Create and push tag $NEW_TAG? [y/N] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    print_warn "Cancelled by user."
    exit 0
fi

# Create annotated tag
print_info "Creating annotated tag $NEW_TAG..."
git tag -a "$NEW_TAG" -m "Release $NEW_TAG"

# Push tag to remote
print_info "Pushing tag $NEW_TAG to origin..."
git push origin "$NEW_TAG"

print_info "Done! GitHub Actions workflow will be triggered automatically."
print_info "Check the release at: https://github.com/TheTitanrain/kerioMirrorGo/releases"
