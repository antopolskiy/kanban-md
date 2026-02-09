# Assets

Scripts for generating screenshots and demo recordings used in the README and landing page.

## Prerequisites

```bash
brew install charmbracelet/tap/freeze   # Static terminal screenshots (PNG/SVG)
brew install asciinema                  # Terminal session recording
brew install agg                        # Convert .cast to .gif
# Optional alternative to asciinema+agg:
brew install charmbracelet/tap/vhs      # Animated terminal GIF recorder
```

## Regenerating screenshots

Build the binary first:

```bash
go build -o /tmp/kanban-md ./cmd/kanban-md
```

### CLI screenshot (Freeze)

```bash
bash assets/cli-screenshot.sh | freeze -o assets/cli-screenshot.png \
  --font.size 14 --theme "Catppuccin Mocha" --padding 20 --window
```

### TUI screenshot (Freeze)

Uses `cmd/tui-showcase` which renders the board with ANSI256 colors forced on:

```bash
bash assets/tui-screenshot.sh
# or manually:
go run ./cmd/tui-showcase | freeze -o assets/tui-screenshot.png \
  --language ansi --font.size 14 --theme "Catppuccin Mocha" --padding 20 --window
```

### Animated demo GIF

**Option A — asciinema + agg** (current approach, better font rendering):

```bash
asciinema rec assets/demo.cast --cols 80 --rows 20 \
  --command "bash assets/demo-record.sh" --overwrite

agg assets/demo.cast assets/demo.gif \
  --font-size 16 --theme dracula --idle-time-limit 3 --last-frame-duration 3
```

**Option B — VHS** (simpler, single tool):

```bash
vhs assets/demo.tape
```
