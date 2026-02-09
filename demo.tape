# kanban-md demo — agent-first file-based kanban
# Records a ~25s GIF showing the core CLI workflow

Output assets/demo.gif

Require bash

Set Shell "bash"
Set FontSize 15
Set FontFamily "JetBrains Mono"
Set LetterSpacing 0
Set Width 960
Set Height 540
Set Padding 20
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful
Set TypingSpeed 30ms
Set PlaybackSpeed 1.5

# Setup: create a temp project dir and put our binary on PATH
Hide
Type "export PS1='$ '" Enter
Type "export PATH=/tmp:$PATH" Enter
Type "cd $(mktemp -d)" Enter
Type "clear" Enter
Sleep 500ms
Show

# 1. Init a board
Type "kanban-md init --name 'My Project'" Sleep 200ms
Enter
Sleep 1s

# 2. Create tasks
Type "kanban-md create 'Set up CI pipeline' --priority high --tags devops" Sleep 200ms
Enter
Sleep 800ms

Type "kanban-md create 'Write API docs' --tags docs" Sleep 200ms
Enter
Sleep 800ms

Type "kanban-md create 'Fix login bug' --priority critical --tags backend" Sleep 200ms
Enter
Sleep 800ms

# 3. List tasks (compact for agent-friendly output)
Type "kanban-md list --compact" Sleep 200ms
Enter
Sleep 1.5s

# 4. Agent picks next task atomically
Type "kanban-md pick --claim agent-1 --move in-progress" Sleep 200ms
Enter
Sleep 1.5s

# 5. Show the board summary
Type "kanban-md board" Sleep 200ms
Enter
Sleep 2s

# 6. Move task to done
Type "kanban-md move 3 done --force" Sleep 200ms
Enter
Sleep 1s

# 7. Final list
Type "kanban-md list --compact" Sleep 200ms
Enter
Sleep 2.5s
