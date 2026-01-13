# Release Process and Binary Distribution

## Overview

Claudio now supports automated builds and releases for Windows 10 and macOS platforms. When a new version is tagged, GitHub Actions automatically builds static binaries and publishes them to GitHub Releases.

## Supported Platforms

The following platforms have pre-built static binaries:

- **Windows 10+ (amd64)**: `claudio-windows-amd64.exe`
- **macOS Intel (amd64)**: `claudio-darwin-amd64`
- **macOS Apple Silicon (arm64)**: `claudio-darwin-arm64`

## Downloading Pre-built Binaries

1. Visit the [GitHub Releases page](https://github.com/joshwand/claudio/releases)
2. Download the binary for your platform
3. Make it executable (macOS/Linux):
   ```bash
   chmod +x claudio-darwin-*
   ```
4. Run the install command:
   ```bash
   ./claudio-darwin-amd64 install  # or claudio-darwin-arm64
   # On Windows: claudio-windows-amd64.exe install
   ```

## Automated Release Process

### Triggering a Release

The GitHub Actions workflow is triggered automatically when a version tag is pushed:

```bash
# Create and push a version tag
git tag v1.10.2
git push origin v1.10.2
```

### What Happens Automatically

1. **Build Phase**: Three separate jobs build static binaries for:
   - Windows 10 (amd64)
   - macOS Intel (amd64)
   - macOS Apple Silicon (arm64)

2. **Release Phase**: After all builds complete:
   - A GitHub Release is created with the tag version
   - All three binaries are attached to the release
   - Release notes are automatically generated from commit history

### Build Configuration

The builds use the following settings:

- **CGO_ENABLED=0**: Static binaries with no C dependencies
- **LDFLAGS="-s -w"**: Strip debug symbols to reduce binary size
- **Go Version**: 1.23.4 (matches go.mod requirement)

### Workflow Location

The workflow is defined in `.github/workflows/release.yml`

## Manual Triggering

The workflow can also be manually triggered from the GitHub Actions UI for testing purposes, even without creating a tag.

## Testing Locally

To test cross-compilation locally before releasing:

```bash
# Windows build
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o claudio-windows-amd64.exe .

# macOS Intel build
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o claudio-darwin-amd64 .

# macOS Apple Silicon build
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o claudio-darwin-arm64 .
```

## Binary Sizes

Static binaries are approximately:
- Windows: ~8.5 MB
- macOS Intel: ~8.4 MB
- macOS Apple Silicon: ~8.1 MB

## Future Enhancements

Potential future additions:
- Linux builds (amd64, arm64)
- Checksums for binary verification
- GPG signatures for security
- Homebrew tap for macOS
- Chocolatey package for Windows
