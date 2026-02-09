# VHS tape file for recording an animated demo GIF.
# Alternative to the asciinema+agg approach in demo-record.sh.
#
# Usage:
#   # 1. Build kanban-md to /tmp
#   go build -o /tmp/kanban-md ./cmd/kanban-md
#
#   # 2. Record with VHS
#   vhs assets/demo.tape
#
# Requirements: vhs (brew install charmbracelet/tap/vhs), kanban-md binary at /tmp/kanban-md

Output assets/demo.gif

Require bash

Set Shell "bash"
Set FontSize 14
Set FontFamily "JetBrains Mono"
Set LineHeight 1.2
Set Width 960
Set Height 400
Set Padding 16
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful
Set TypingSpeed 20ms
Set PlaybackSpeed 2

Hide
Type "export PS1='$ '" Enter
Type "export PATH=/tmp:$PATH" Enter
Type "cd $(mktemp -d)" Enter
Type "kanban-md init --name 'Test' >/dev/null 2>&1" Enter
Type "kanban-md create 'Set up CI' --priority high --tags devops >/dev/null 2>&1" Enter
Type "kanban-md create 'Fix login bug' --priority critical --tags backend >/dev/null 2>&1" Enter
Type "clear" Enter
Sleep 300ms
Show

Type "kanban-md list --compact" Sleep 100ms
Enter
Sleep 1s

Type "kanban-md pick --claim agent-1 --move in-progress" Sleep 100ms
Enter
Sleep 1.5s

Type "kanban-md list --compact" Sleep 100ms
Enter
Sleep 2s
