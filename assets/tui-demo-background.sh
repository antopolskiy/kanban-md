#!/bin/bash
# Background script that runs kanban-md commands while the TUI is live.
# Each command modifies task files on disk; the TUI's file watcher
# picks up changes and auto-refreshes (~100ms debounce).
#
# This script runs hidden behind the TUI's alternate screen buffer.
# All output is discarded.
#
# Expects DEMO_KANBAN_DIR env var pointing to the kanban directory.

set -e
export PATH=/tmp:$PATH
K="kanban-md"
D="--dir $DEMO_KANBAN_DIR"

# Wait for TUI to fully render before starting.
sleep 2.5

# --- Phase 1: agents claim tasks and start working ---
#
# Three agents use three different (all valid) workflows:
#   frost-maple: `pick` — find + claim + move in one operation (recommended)
#   amber-swift: two-step claim-in-place then move
#   coral-dusk:  single-step move with --claim

# frost-maple picks the highest-priority unclaimed task (the critical
# security fix) and moves it to in-progress in one step.
$K pick --claim frost-maple --status todo --move in-progress $D >/dev/null 2>&1
sleep 1.5

# amber-swift claims the monitoring task while it's still in todo...
$K edit 5 --claim amber-swift $D >/dev/null 2>&1
sleep 0.8

# ...then moves it to in-progress.
$K move 5 in-progress --claim amber-swift $D >/dev/null 2>&1
sleep 2

# --- Phase 2: work progresses, new agent joins ---

# frost-maple finishes implementation — moves to review.
$K move 6 review --claim frost-maple $D >/dev/null 2>&1
sleep 1.5

# coral-dusk claims the rate limiting task and starts working
# in a single step (move with --claim).
$K move 4 in-progress --claim coral-dusk $D >/dev/null 2>&1
sleep 2

# --- Phase 3: tasks flow to done ---
#
# Per the kanban-based-development workflow, agents release their
# claim before moving to done (edit --release, then move done).

# frost-maple's review passes — release claim, then done.
$K edit 6 --release $D >/dev/null 2>&1
sleep 0.5
$K move 6 done $D >/dev/null 2>&1
sleep 1.5

# amber-swift finishes — moves to review.
$K move 5 review --claim amber-swift $D >/dev/null 2>&1
sleep 1.5

# coral-dusk finishes — moves to review.
$K move 4 review --claim coral-dusk $D >/dev/null 2>&1
sleep 1.5

# amber-swift's review passes — release claim, then done.
$K edit 5 --release $D >/dev/null 2>&1
sleep 0.5
$K move 5 done $D >/dev/null 2>&1
sleep 1.5

# coral-dusk's review passes — release claim, then done.
$K edit 4 --release $D >/dev/null 2>&1
sleep 0.5
$K move 4 done $D >/dev/null 2>&1

# Hold final state before the record script ends.
sleep 2
