# Future request corpus for kanban-md

Date: 2026-09-20

## Method

These are invented requests for product review exercises. They are not customer interviews, live issues, commitments, or claims that the requested behavior exists. Each case gives a requester's situation and constraints without supplying a recommended decision.

The capability baseline comes from [README.md](../../README.md), [the task model](../../internal/task/task.go), [task file reading and writing](../../internal/task/file.go), [configuration](../../internal/config/config.go), [list commands](../../cmd/list.go), and [task selection](../../internal/board/pick.go). The current product has Markdown task files with defined frontmatter fields, configurable statuses and priorities, CLI filters and output formats, a TUI, task relationships, and cooperative claims. Some fictional requesters may be overlooking an existing capability. No live issue tracker, design-principle research report, product-review skill, or other agent's work informed this corpus.

## F01: Run a film festival submission board

> "Could you add a film festival mode? We need received, screening, shortlisted, and decided columns, with routine and urgent priorities. Nobody here is a coding agent, and I don't want volunteers entering claim names just to move a film."

The requester coordinates six volunteers who review 90 short films. They work at a shared office computer and keep screening notes in each task's Markdown body. One technically comfortable volunteer installed kanban-md because the group already exchanges Markdown notes.

They need the chosen column order to control next and previous moves. A decided film should stop appearing as unfinished work. They can edit one configuration file during setup, but ordinary volunteers will use the TUI. They do not need scoring, video playback, submission payments, or an online application form. They would prefer to keep using the standard binary.

## F02: Find a research note from the TUI

> "I remember putting 'copper contamination' in a task, but searching the board finds nothing unless those words are in the title. Can the TUI search the notes and tags too, and tell me why a card matched?"

A materials researcher has 280 experiment tasks. Titles use specimen codes, while useful descriptions and instrument names live in the body and tags. They already get the expected results from `kanban-md list --search`, then return to the TUI to inspect and move tasks.

Their laptop is often offline in the lab. They expect literal, case-insensitive matching and want ticket-ID lookup to remain usable. They do not want a model to summarize or reinterpret the query. A short indication that the match came from a body or tag would be enough; displaying every matching paragraph on the board would make the cards hard to scan.

## F03: Repeat food bank opening checks

> "Please let me make 'Check refrigerator temperatures' recur every Monday, Wednesday, and Friday. Completing today's check should leave its record intact and prepare the next one."

A food bank supervisor currently copies a task by hand for each opening day. A check has an assignee, a short checklist, and a due date. The building closes on public holidays, and sometimes a missed check is recorded the following morning.

They need separate records of what happened on each date. A missed occurrence must remain visible instead of silently becoming the next occurrence. They want to skip a holiday without changing the future pattern, and they do not want months of future cards filling the board. The office computer may be switched off overnight. Running a command when opening the board is acceptable; relying on someone remembering to copy the task is the problem they want solved.

## F04: Review commitments across separate clients

> "I have a board for every translation client. Can I open one view of everything due this week, then jump into the right client's task without merging the boards?"

A freelance translator maintains eight directories, each with its own board. Several boards contain task 12, and clients use different status names. The translator currently visits each directory to build a morning work list and sometimes overlooks a deadline.

The combined view must identify the originating board alongside each task ID. It must not copy client notes into a ninth board, and an unavailable directory should be distinguishable from a board with no due work. They only need to inspect tasks from the combined view initially. Their client directories remain independently movable and may use different configuration versions after an upgrade. They can supply an explicit list of directories.

## F05: Keep museum metadata through ordinary edits

> "Our collection script adds `accession_number` and a nested `storage` value to task frontmatter. Moving a task with kanban-md shouldn't remove those fields. Can the two tools share the file?"

A small museum tracks conservation work on a board. An existing script reads the same Markdown files to associate tasks with object records. A conservator changes task status and assignee through the CLI; the collection script owns the extra metadata.

The museum is not asking kanban-md to understand accession numbers, validate storage locations, or display those values in every view. The values must survive title changes and status moves, including nested mappings and lists. The collection script can adopt a dedicated metadata namespace if necessary. Existing files lack that namespace, and staff need a predictable way to handle them during an upgrade. Keeping YAML comments would be convenient, but preserving the actual values matters more.

## F06: Put garden deadlines on a shared calendar

> "Our planting and delivery dates are on the board, but volunteers look at their calendars. Can we publish those due dates into a calendar they can subscribe to?"

A community garden has about 60 active tasks, each with a date rather than a time of day. A coordinator updates the board locally once or twice a week and already has a small website where they can upload a generated file.

Calendar entries should link back to enough information to identify the task, and rescheduling a task should update its existing entry instead of creating a duplicate. Completed tasks should have a defined treatment. The shared calendar may contain task titles and dates, but not private notes or assignee contact details. Volunteers use different calendar products. The coordinator does not need calendar edits to flow back into the board and cannot administer a continuously running server.

## F07: Use the board from a phone terminal

> "Can you add a phone layout? When I connect over SSH, I want one column across the whole screen and a way to switch columns. I also want longer task titles instead of several tiny columns."

A community theatre stage manager checks a backstage setup board from a phone while walking around the venue. Their terminal is usually 45 characters wide. They use arrow keys supplied by the SSH app and rarely have a physical keyboard attached.

The board has seven columns and task titles up to 70 characters. The stage manager wants to inspect and move existing tasks, not write long descriptions on the phone. They can add a flag to their saved SSH command or ask the volunteer administrator to change a board setting. They need the same board to remain comfortable on the office monitor. They are not asking for a native mobile application or offline phone editing.

## F08: Treat a missing prerequisite differently

> "A task with `depends_on: [73]` became eligible after task 73 disappeared during a bad file copy. Can this board refuse to pick work with a missing prerequisite and explain what's missing?"

A software team uses `pick` in an agent loop. Their task directory travels between working copies through an internal script, and an incomplete copy briefly omitted a dependency. They want to catch this before an agent begins dependent work.

Other boards in the organization deliberately remove old completed task files, so changing every board's behavior would disrupt them. The affected team can enable a board setting and repair existing references before doing so. A missing dependency should remain distinct from one that exists but is unfinished. They need consistent answers from the eligibility listing and `pick`, and a diagnostic that helps a human find the faulty reference.

## F09: Dispatch idle coding agents automatically

> "Can kanban-md keep three coding agents busy overnight? It should start an agent for the next task, give it a worktree, notice when it stalls, and choose a replacement task without me driving each run."

A maintainer already runs agents manually with the bundled workflow. The board has priorities, dependencies, claims, and handoff notes, but an agent process exiting still requires the maintainer to inspect the result and launch another session.

They use two different agent CLIs and want to retain that choice. A session must stop when it needs a human decision or reaches a spending limit. The maintainer wants a morning account of processes started, work completed, and unresolved failures. Pull requests may be prepared automatically, while merging and deploying remain manual. They can leave a workstation running, but would rather not maintain a separate orchestration application alongside the board.

## F10: Reconcile a board after disconnected fieldwork

> "Two people added tasks while their laptops were offline, and syncing the folder gave us duplicate numbers. Can the board help reconcile the copies without breaking references or pretending that an old claim still owns the work?"

An ecology group uses a folder synchronization service for survey preparation. Fieldworkers sometimes spend two days without a connection. They add observations as tasks and refer to existing tasks by ID in dependencies and plain body text. On return, the sync service may retain both conflicting file versions.

The group accepts that two people can accidentally start the same work while disconnected. They want a reviewable reconciliation result when the machines reconnect, including the old and new task identities. They cannot change the sync service or assume an always-reachable central machine. A local repair command is acceptable, but silently rewriting prose references or throwing away either person's notes would be difficult to trust.

## F11: Produce a meeting packet without the terminal

> "Can I export a weekly board packet for our trustees? It should show unfinished work by assignee, overdue dates, and what finished this week, in something we can print and read without installing kanban-md."

A literacy charity's administrator maintains the board on a laptop. Trustees receive a packet before a monthly meeting and annotate it on paper. The administrator currently copies terminal output into a word processor and fixes the alignment by hand.

They need a snapshot with a clear generation date, not a live portal. The packet must omit task bodies because those sometimes contain donor details. It should use the board's own status names and keep long titles readable. Markdown or HTML that their existing office tools can convert to PDF is acceptable. The administrator wants to rerun the same command next month without rebuilding the report layout or maintaining a custom script.

## F12: Finish a parent when its chapters are finished

> "When every chapter task is done, can the book's parent task move to done automatically? And if I reopen a chapter, can it bring the book back into progress?"

An author organizes a nonfiction book as one parent task with twelve chapter tasks. The current child progress display helps, but they have forgotten to update the parent after the last chapter finished. Other parent tasks on the board represent submissions that need a separate acceptance decision.

The author wants this behavior only for parents they choose. Archived chapters sometimes mean dropped material, not completed writing. A book may also have nested section tasks under each chapter. The requester wants a way to understand why a parent changed and avoid losing a deliberate status such as waiting-for-publisher. They have not decided whether finishing direct children or every descendant should count as finishing the book.

## F13: Share work with a team that stays in GitHub

> "I prefer the local board, but the rest of our team works in GitHub Issues. Can I link selected tasks and keep their titles, descriptions, and open or closed state in sync both ways?"

An open-source maintainer uses local claims and dependencies to coordinate agents. Contributors file issues and edit descriptions in GitHub. Copying updates manually has left local tasks claiming that bugs are still open after contributors closed the corresponding issues.

Only explicitly linked tasks should synchronize. Claim names, internal handoff notes, and local dependencies should remain local. The maintainer expects a visible conflict when both sides change a description between syncs and wants a preview before publishing changes. GitHub authentication is already available through their CLI. The local board should remain usable during an outage, and other projects that never use GitHub should require no account or setup.

## F14: Require evidence before accepting a repair

> "Could a task refuse to enter inspected unless it has a test note and the inspector's name? We keep accepting repair jobs with one of those missing."

A volunteer repair workshop uses intake, repairing, inspected, and returned columns. Technicians record measurements and short observations in task bodies. A coordinator uses the CLI for batches, while bench volunteers use the TUI. Some tasks are administrative work and need no inspection.

The workshop wants to prevent accidental omissions through either interface. It does not need to defend against someone deliberately editing a Markdown file to bypass the rule. Staff want the error to say what is missing and preserve the previous task state. Existing completed repairs should remain readable without retroactive cleanup. Inspection requirements may vary by task tag, and a supervisor occasionally needs to record an explicit exception with a reason.
