#!/bin/bash
# Demo recording script for asciinema + agg.
# Produces a .cast file that agg converts to a GIF.
#
# Usage (from repo root):
#   bash assets/demo-gen.sh
#
# Requirements: asciinema, agg, python3, kanban-md binary at /tmp/kanban-md

set -e
export PATH=/tmp:$PATH
# Force 256-color output even without a real TTY.
export CLICOLOR_FORCE=1
export COLORFGBG="15;0"
cd "$(mktemp -d)"

# Silent setup — create board and tasks before recording starts
kanban-md init --name "My Project" >/dev/null 2>&1
kanban-md create "Fix login bypass" --priority critical --tags security >/dev/null 2>&1
kanban-md create "Add rate limiter" --priority high --tags backend >/dev/null 2>&1
kanban-md create "Write API docs" --priority medium --tags docs >/dev/null 2>&1

# 1. Show initial board — three tasks, all unclaimed
printf '\033[38;5;75m$\033[0m kanban-md list\n'
sleep 0.3
kanban-md list 2>/dev/null
sleep 1.5
echo

# 2. Agent 1 picks — gets the critical task
printf '\033[38;5;75m$\033[0m kanban-md pick --claim frost-maple --move in-progress\n'
sleep 0.3
kanban-md pick --claim frost-maple --move in-progress 2>/dev/null
sleep 1.5
echo

# 3. Agent 2 picks — skips claimed task, gets the next one
printf '\033[38;5;75m$\033[0m kanban-md pick --claim amber-swift --move in-progress\n'
sleep 0.3
kanban-md pick --claim amber-swift --move in-progress 2>/dev/null
sleep 1.5
echo

# 4. Final state — two agents, no conflicts
printf '\033[38;5;75m$\033[0m kanban-md list\n'
sleep 0.3
kanban-md list 2>/dev/null
sleep 3
