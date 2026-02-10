#!/bin/bash
# One-command demo GIF generator.
# Usage: bash assets/demo-gen.sh
# Requirements: asciinema, agg, python3
set -e

echo "Building kanban-md..."
go build -o /tmp/kanban-md ./cmd/kanban-md

echo "Recording demo..."
asciinema rec assets/demo.cast --cols 80 --rows 24 \
  --command "bash assets/demo-record.sh" --overwrite

echo "Capping OSC query delays..."
python3 -c "
import json
lines = open('assets/demo.cast').readlines()
out = open('assets/demo-capped.cast', 'w')
for line in lines:
    try:
        ev = json.loads(line)
        if isinstance(ev, list) and ev[0] > 1.5:
            ev[0] = 0.3
        out.write(json.dumps(ev) + '\n')
    except (json.JSONDecodeError, TypeError):
        out.write(line)
out.close()
"
mv assets/demo-capped.cast assets/demo.cast

echo "Converting to GIF..."
agg assets/demo.cast assets/demo.gif \
  --font-size 16 --theme dracula --idle-time-limit 3 --last-frame-duration 3

echo "Wrote assets/demo.gif"
