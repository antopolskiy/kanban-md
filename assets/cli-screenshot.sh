#!/bin/bash
# Generate CLI screenshot using Charmbracelet Freeze.
#
# Usage:
#   # 1. Build kanban-md to /tmp
#   go build -o /tmp/kanban-md ./cmd/kanban-md
#
#   # 2. Run the script and pipe to freeze
#   bash assets/cli-screenshot.sh | freeze -o assets/cli-screenshot.png \
#     --font.size 14 --theme "Catppuccin Mocha" --padding 20 --window
#
# Requirements: freeze (brew install charmbracelet/tap/freeze), kanban-md binary at /tmp/kanban-md

set -e
K="/tmp/kanban-md"
DIR=$(mktemp -d)
cd "$DIR"

# Silent setup
$K init --name "My Project" >/dev/null 2>&1
$K create "Set up CI pipeline" --priority high --tags devops >/dev/null 2>&1
$K create "Write API docs" --tags docs >/dev/null 2>&1
$K create "Fix login bug" --priority critical --tags backend >/dev/null 2>&1

# Show commands and output
printf '\033[1;32m$\033[0m kanban-md list --compact --status todo,in-progress,review\n'
$K list --compact --status todo,in-progress,review
echo ""
printf '\033[1;32m$\033[0m kanban-md pick --claim sage-river --status todo --move in-progress\n'
$K pick --claim sage-river --status todo --move in-progress
