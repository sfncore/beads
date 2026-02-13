# Beads Scripts

Utility scripts for maintaining the beads project.

## release.sh (⭐ The Easy Button)

**One-command release** from version bump to local installation.

### Usage

```bash
# Full release (does everything)
./scripts/release.sh 0.9.3

# Preview what would happen
./scripts/release.sh 0.9.3 --dry-run
```

### What It Does

This master script automates the **entire release process**:

1. ✅ Kills running daemons (avoids version conflicts)
2. ✅ Runs tests and linting
3. ✅ Bumps version in all files
4. ✅ Commits and pushes version bump
5. ✅ Creates and pushes git tag
6. ✅ Updates Homebrew formula
7. ✅ Upgrades local brew installation
8. ✅ Verifies everything works

**After this script completes, your system is running the new version!**

### Examples

```bash
# Release version 0.9.3
./scripts/release.sh 0.9.3

# Preview a release (no changes made)
./scripts/release.sh 1.0.0 --dry-run
```

### Prerequisites

- Clean git working directory
- All changes committed
- golangci-lint installed
- Homebrew installed (for local upgrade)
- Push access to steveyegge/beads

### Output

The script provides colorful, step-by-step progress output:
- 🟨 Yellow: Current step
- 🟩 Green: Step completed
- 🟥 Red: Errors
- 🟦 Blue: Section headers

### What Happens Next

After the script finishes:
- GitHub Actions builds binaries for all platforms (~5 minutes)
- PyPI package is published automatically
- Users can `brew upgrade beads` to get the new version
- GitHub Release is created with binaries and changelog

---

## bump-version.sh

Bumps the version number across all beads components in a single command.

### Usage

```bash
# Show usage
./scripts/bump-version.sh

# Update versions (shows diff, no commit)
./scripts/bump-version.sh 0.9.3

# Update versions and auto-commit
./scripts/bump-version.sh 0.9.3 --commit
```

### What It Does

Updates version in all these files:
- `cmd/bd/version.go` - bd CLI version constant
- `claude-plugin/.claude-plugin/plugin.json` - Plugin version
- `.claude-plugin/marketplace.json` - Marketplace plugin version
- `integrations/beads-mcp/pyproject.toml` - MCP server version
- `README.md` - Alpha status version
- `PLUGIN.md` - Version requirements

### Features

- **Validates** semantic versioning format (MAJOR.MINOR.PATCH)
- **Verifies** all versions match after update
- **Shows** git diff of changes
- **Auto-commits** with standardized message (optional)
- **Cross-platform** compatible (macOS and Linux)

### Examples

```bash
# Bump to 0.9.3 and review changes
./scripts/bump-version.sh 0.9.3
# Review the diff, then manually commit

# Bump to 1.0.0 and auto-commit
./scripts/bump-version.sh 1.0.0 --commit
git push origin main
```

### Why This Script Exists

Previously, version bumps only updated `cmd/bd/version.go`, leaving other components out of sync. This script ensures all version numbers stay consistent across the project.

### Safety

- Checks for uncommitted changes before proceeding
- Refuses to auto-commit if there are existing uncommitted changes
- Validates version format before making any changes
- Verifies all versions match after update
- Shows diff for review before commit

---

## sign-windows.sh

Signs Windows executables with an Authenticode certificate using osslsigncode.

### Usage

```bash
# Sign a Windows executable
./scripts/sign-windows.sh path/to/bd.exe

# Environment variables required for signing:
export WINDOWS_SIGNING_CERT_PFX_BASE64="<base64-encoded-pfx>"
export WINDOWS_SIGNING_CERT_PASSWORD="<certificate-password>"
```

### What It Does

This script is called automatically by GoReleaser during the release process:

1. **Decodes** the PFX certificate from base64
2. **Signs** the Windows executable using osslsigncode
3. **Timestamps** the signature using DigiCert's RFC3161 server
4. **Replaces** the original binary with the signed version
5. **Verifies** the signature was applied correctly

### Prerequisites

- `osslsigncode` installed (`apt install osslsigncode` or `brew install osslsigncode`)
- EV code signing certificate exported as PFX file
- GitHub secrets configured:
  - `WINDOWS_SIGNING_CERT_PFX_BASE64` - base64-encoded PFX file
  - `WINDOWS_SIGNING_CERT_PASSWORD` - certificate password

### Graceful Degradation

If the signing secrets are not configured:
- The script prints a warning and exits successfully
- GoReleaser continues without signing
- The release proceeds with unsigned Windows binaries

This allows releases to work before a certificate is acquired.

### Why This Script Exists

Windows code signing helps reduce antivirus false positives that affect Go binaries.
Kaspersky and other AV software commonly flag unsigned Go executables as potentially
malicious due to heuristic detection. See `docs/ANTIVIRUS.md` for details.

---

## generate-workspace.sh

Generates a VS Code multi-root workspace that includes all Git worktrees as separate folders.

### Usage

```bash
# Using beads CLI (recommended)
bd workspace

# Or run script directly
./scripts/generate-workspace.sh
```

### What It Does

1. **Discovers** all Git worktrees using `git worktree list --porcelain`
2. **Validates** each worktree directory exists and is accessible
3. **Generates** VS Code workspace JSON with:
   - Each worktree as a named folder
   - Branch names in folder titles for easy identification
   - Optimized settings for Git workflow
   - Recommended extensions (GitLens, Git Graph, etc.)
4. **Opens** workspace in VS Code if `code` command is available

### Output File

Creates `beads-worktrees.code-workspace` in repository root:
```json
{
    "folders": [
        {
            "name": "gt (main)",
            "path": "."
        },
        {
            "name": "beads-metadata (beads-metadata)", 
            "path": ".git/beads-worktrees/beads-metadata"
        }
    ],
    "settings": {
        "git.enableSmartCommit": true,
        "files.exclude": {
            "**/.git": true
        }
    },
    "extensions": {
        "recommendations": [
            "ms-vscode.vscode-git-graph",
            "eamodio.gitlens"
        ]
    }
}
```

### Benefits

- ✅ **No context switching** - All branches open simultaneously
- ✅ **Independent debugging** - Each worktree has its own VS Code context
- ✅ **Easy comparison** - Side-by-side file comparison across branches
- ✅ **Optimized for beads** - Settings tuned for Git worktree workflow

### Examples

```bash
# Generate workspace for current worktrees
./scripts/generate-workspace.sh

# Output shows found worktrees and opens VS Code
[INFO] Discovering worktrees...
[INFO] Found 3 worktree(s)
[INFO]   ✓ gt -> main
[INFO]   ✓ beads-metadata -> beads-metadata
[INFO]   ✓ beads-sync -> beads-sync
[SUCCESS] Workspace file created: /home/ubuntu/gt/beads-worktrees.code-workspace
```

---

## watch-workspaces.sh

Watches for worktree changes and auto-updates VS Code workspace.

### Usage

```bash
# Start watching for worktree changes
./scripts/watch-workspaces.sh

# Ctrl+C to stop
```

### What It Does

1. **Monitors** Git worktree state using hash comparison
2. **Auto-updates** workspace when worktrees are added/removed
3. **Runs** every 5 seconds (configurable via `WATCH_INTERVAL`)
4. **Preserves** VS Code session by regenerating in-place

### Features

- **Real-time updates** - No manual regeneration needed
- **Lightweight** - Uses hash comparison to minimize processing
- **Auto-initialization** - Creates initial workspace if missing
- **Cross-platform** - Works on macOS, Linux, and Windows

### Configuration

```bash
# Custom watch interval (seconds)
WATCH_INTERVAL=10 ./scripts/watch-workspaces.sh

# Disable auto-opening when workspace updates
AUTO_OPEN=false ./scripts/watch-workspaces.sh
```

### Workflow Integration

Perfect for beads development:
```bash
# Terminal 1: Start watcher
./scripts/watch-workspaces.sh

# Terminal 2: Create new worktree
bd worktree create feature/new-auth --branch feature/user-auth

# Workspace auto-updates in ~5 seconds
# VS Code shows new folder: "feature-new-auth (feature/user-auth)"
```

---

## Future Scripts

Additional maintenance scripts may be added here as needed.
