#!/bin/bash
# Generate VS Code workspace for all Git worktrees

set -euo pipefail

# Configuration
WORKSPACE_NAME="beads-worktrees.code-workspace"
MAIN_REPO="$(git rev-parse --show-toplevel)"
WORKSPACE_FILE="$MAIN_REPO/$WORKSPACE_NAME"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}
print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}
print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}
print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Ensure we're in a git repo
if ! git rev-parse --git-dir >/dev/null 2>&1; then
    print_error "Not in a Git repository"
    exit 1
fi

# Get worktree and crew information
print_status "Discovering worktrees..."

# Use --porcelain for machine-readable output
worktrees_output=$(git worktree list --porcelain)

# Get crew workspaces
print_status "Discovering crew workspaces..."
crews_output=$(gt crew list --all --json 2>/dev/null || echo '{"crews":[]}')

# Parse worktrees into arrays
paths=()
branches=()
commits=()

# Parse crews into arrays
crew_paths=()
crew_names=()
crew_branches=()
crew_git_status=()

current_worktree=""
current_branch=""
current_commit=""

while IFS= read -r line; do
    case "$line" in
        worktree*)
            # Save previous worktree if exists
            if [[ -n "$current_worktree" ]]; then
                paths+=("$current_worktree")
                branches+=("$current_branch")
                commits+=("$current_commit")
            fi
            # Start new worktree
            current_worktree="${line#worktree }"
            current_branch=""
            current_commit=""
            ;;
        branch*)
            current_branch="${line#branch refs/heads/}"
            # Handle detached HEAD case
            if [[ "$current_branch" == "$line" ]]; then
                current_branch="detached"
            fi
            ;;
        HEAD*)
            current_commit="${line#HEAD }"
            ;;
        # Empty line indicates end of worktree entry
        "")
            if [[ -n "$current_worktree" ]]; then
                paths+=("$current_worktree")
                branches+=("$current_branch")
                commits+=("$current_commit")
                current_worktree=""
            fi
            ;;
    esac
done <<< "$worktrees_output"

# Handle last worktree if no empty line at end
if [[ -n "$current_worktree" ]]; then
    paths+=("$current_worktree")
    branches+=("$current_branch")
    commits+=("$current_commit")
fi

# Parse crew workspaces from JSON (manual approach)
if [[ -n "$crews_output" ]]; then
    # Extract crew info using python for reliable JSON parsing
    python3 -c "
import json
import sys

try:
    data = json.load(sys.stdin)
    for crew in data:
        name = crew.get('name', '')
        branch = crew.get('branch', '')
        git_clean = crew.get('git_clean', False)
        git_status = 'clean' if git_clean else 'dirty'
        path = crew.get('path', '')
        
        if name and path and branch:
            print(f'{name}|{branch}|{git_status}|{path}')
except:
    pass
" <<< "$crews_output" | while IFS='|' read -r crew_name crew_branch crew_git_status crew_path; do
        if [[ -n "$crew_name" && -n "$crew_path" && -d "$crew_path" ]]; then
            crew_paths+=("$crew_path")
            crew_names+=("$crew_name")
            crew_branches+=("$crew_branch")
            crew_git_status+=("$crew_git_status")
            
            print_status "  ✓ $crew_name -> $crew_branch (Git: $crew_git_status)"
        fi
    done
fi

# Combine worktrees and crews
total_items=${#paths[@]}
if [[ $total_items -eq 0 && ${#crew_paths[@]} -eq 0 ]]; then
    print_error "No worktrees or crew workspaces found"
    exit 1
fi

if [[ $total_items -gt 0 ]]; then
    print_status "Found $total_items worktree(s)"
fi
if [[ ${#crew_paths[@]} -gt 0 ]]; then
    print_status "Found ${#crew_paths[@]} crew workspace(s)"
fi

# Validate worktree directories and filter out internal beads worktrees
valid_paths=()
valid_branches=()

for i in "${!paths[@]}"; do
    path="${paths[$i]}"
    branch="${branches[$i]}"
    
    # Skip internal beads worktrees (used for sync-branch operations)
    if [[ "$path" == *".git/beads-worktrees"* ]]; then
        print_status "  - $(basename "$path") -> $branch (internal beads worktree, skipped)"
        continue
    fi
    
    if [[ -d "$path" ]]; then
        valid_paths+=("$path")
        valid_branches+=("$branch")
        print_status "  ✓ $(basename "$path") -> $branch"
    else
        print_warning "  ✗ $(basename "$path") -> $branch (directory missing)"
    fi
done

# Validate crew workspaces
for i in "${!crew_paths[@]}"; do
    path="${crew_paths[$i]}"
    name="${crew_names[$i]}"
    branch="${crew_branches[$i]}"
    git_status="${crew_git_status[$i]}"
    
    if [[ -d "$path" ]]; then
        valid_paths+=("$path")
        valid_branches+=("$branch")
        print_status "  ✓ $name -> $branch (Git: $git_status)"
    else
        print_warning "  ✗ $name -> $branch (directory missing)"
    fi
done

# Add potential agent session directories (specific directories where agents edit code)
print_status "Scanning for agent session directories..."

# Look ONLY for specific agent session directories, not all code directories
agent_dirs=()

# Specific agent session patterns (very targeted)
session_patterns=(
    "gastown"          # Agent session directories mentioned in beads
    "agent*"            # Direct agent session directories
    "session*"           # Session directories  
    "workspace*"         # Agent workspace directories
    "crew*"              # Crew work directories
    "rig*"               # Rig work directories
)

# Only find directories that match agent session patterns
for pattern in "${session_patterns[@]}"; do
    while IFS= read -r found_dir; do
        if [[ -d "$found_dir" ]]; then
            # Skip if already included as worktree
            skip=false
            for existing in "${valid_paths[@]}"; do
                if [[ "$found_dir" == "$existing" ]]; then
                    skip=true
                    break
                fi
            done
            
            if [[ "$skip" == "false" ]]; then
                # Additional check: only include if this looks like an active agent workspace
                # Check for signs of active work: recent files, agent configs, etc.
                if [[ -f "$found_dir/.agent" ]] || [[ -f "$found_dir/beads.db" ]] || find "$found_dir" -maxdepth 1 -name "*.json" -o -name "*.yaml" -o -name "*.toml" 2>/dev/null | head -1 | grep -q .; then
                    agent_dirs+=("$found_dir")
                    print_status "  ✓ Found agent session directory: $(basename "$found_dir")"
                fi
            fi
        fi
    done < <(find "$MAIN_REPO" -maxdepth 2 -name "$pattern" -type d 2>/dev/null || true)
done

# Add found directories to workspace
for agent_dir in "${agent_dirs[@]}"; do
    valid_paths+=("$agent_dir")
    valid_branches+=("code-directory")
    print_status "  + Adding code directory: $(basename "$agent_dir")"
done

if [[ ${#valid_paths[@]} -eq 0 ]]; then
    print_error "No valid worktree or crew directories found"
    exit 1
fi

# Generate workspace JSON
print_status "Generating workspace configuration..."

# Start JSON
cat > "$WORKSPACE_FILE" << 'EOF'
{
    "folders": [
EOF

# Add each worktree as a folder
for i in "${!valid_paths[@]}"; do
    path="${valid_paths[$i]}"
    branch="${valid_branches[$i]}"
    name=$(basename "$path")
    
    # Convert to relative path from main repo
    rel_path=$(realpath --relative-to="$MAIN_REPO" "$path")
    
    # Add comma for all but last
    comma=$([[ $i -lt $((${#valid_paths[@]} - 1)) ]] && echo "," || echo "")
    
    # Generate folder configuration with branch info
    cat >> "$WORKSPACE_FILE" << EOF
        {
            "name": "$name ($branch)",
            "path": "$rel_path"
        }$comma
EOF
done

# Close JSON
cat >> "$WORKSPACE_FILE" << 'EOF'
    ],
    "settings": {
        "git.enableSmartCommit": true,
        "git.autofetch": true,
        "git.confirmSync": false,
        "git.showInlineOpenFileAction": false,
        "git.suggestSmartCommit": false,
        "files.exclude": {
            "**/.git": true,
            "**/.DS_Store": true,
            "**/Thumbs.db": true
        },
        "search.exclude": {
            "**/.git": true,
            "**/node_modules": true,
            "**/bower_components": true
        }
    },
    "extensions": {
        "recommendations": [
            "ms-vscode.vscode-git-graph",
            "eamodio.gitlens",
            "donjayamanne.githistory"
        ]
    }
}
EOF

print_success "Workspace file created: $WORKSPACE_FILE"

# Optional: Open workspace in VS Code if code command is available
if command -v code >/dev/null 2>&1; then
    echo
    print_status "Opening workspace in VS Code..."
    code "$WORKSPACE_FILE"
else
    echo
    print_status "To open the workspace, run:"
    echo "  code '$WORKSPACE_FILE'"
fi

# Show workspace summary
echo
print_status "Workspace Summary:"
for i in "${!valid_paths[@]}"; do
    path="${valid_paths[$i]}"
    branch="${valid_branches[$i]}"
    name=$(basename "$path")
    echo "  • $name ($branch)"
done

print_success "Done! All ${#valid_paths[@]} worktrees are now available in one VS Code workspace."