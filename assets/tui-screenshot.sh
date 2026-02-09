#!/bin/bash
# Generate TUI screenshot using cmd/tui-showcase + Charmbracelet Freeze.
#
# The tui-showcase program renders the board with ANSI256 colors forced on,
# producing colored output that Freeze converts to a PNG.
#
# Usage (from repo root):
#   bash assets/tui-screenshot.sh
#
# Requirements: freeze (brew install charmbracelet/tap/freeze)

set -e
go run ./cmd/tui-showcase | freeze -o assets/tui-screenshot.png \
  --language ansi --font.size 14 --theme "Catppuccin Mocha" --padding 20 --window
echo "Wrote assets/tui-screenshot.png"
