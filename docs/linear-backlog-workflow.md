# Linear Backlog Workflow

End-to-end guide for managing Linear backlogs with `cadre linear` (CLI) and `cadre sprint` (orchestration).

## Overview

The workflow connects two tools:

1. **`cadre linear`** — CLI commands for querying and managing Linear issues, cycles, labels, and projects
2. **`cadre sprint`** — Multi-agent orchestration that executes work from a sprint config YAML

The bridge between them is a **sprint config file** that maps Linear issues to agent workstreams.

## 1. Query the Backlog

Use `cadre linear` to explore what needs to be done.

```bash
# List all issues
cadre linear issues list

# Filter by state
cadre linear issues list --state "Todo"
cadre linear issues list --state "In Progress"

# Filter by label
cadre linear issues list --label "type:bug"
cadre linear issues list --label "area:api"

# Filter by cycle
cadre linear issues list --cycle S02

# Filter by project
cadre linear issues list --project "Feature Completeness"

# Combine filters
cadre linear issues list --state "Todo" --label "type:feature" --cycle S02

# Get issue details
cadre linear issues get RAC-127
```

## 2. Plan the Sprint

Select issues and group them into workstreams. Each workstream maps to one Claude Code agent.

Guidelines for workstream design:
- **Group related issues** — issues that touch the same subsystem go in one workstream
- **One branch per workstream** — each agent works on its own git branch
- **Assign a crew type** — match the work to an agent profile (security-hardening, feature-development, quick-fix, etc.)
- **Keep workstreams independent** — minimize cross-workstream dependencies for parallel execution

Example planning:

```
RAC-127 (session hardening)  ─┐
RAC-209 (token rotation)     ─┼─ workstream: sec-session (security-hardening)
RAC-210 (session audit)      ─┘

RAC-128 (websocket auth)     ─┐
RAC-211 (ws rate limiting)   ─┼─ workstream: sec-websocket (security-hardening)
RAC-212 (ws audit logging)   ─┘

RAC-187 (operator reg)       ─┐
RAC-188 (healthcheck fix)    ─┼─ workstream: controller-ops (quick-fix)
RAC-189 (metrics endpoint)   ─┘

RAC-169 (portal dashboard)   ─── workstream: ui-portal (feature-development)
```

## 3. Write the Sprint Config

Create a YAML file in `.cadre-workspace/sprints/` or `sprints/`.

```yaml
# .cadre-workspace/sprints/S02.yaml
name: S02
description: "Security & Operator Registration"

workstreams:
  - name: sec-session
    crew: security-hardening
    issues: [RAC-127, RAC-209, RAC-210]
    branch: agent-1-sec-session
    working_dir: /path/to/repo  # optional, defaults to CWD

  - name: sec-websocket
    crew: security-hardening
    issues: [RAC-128, RAC-211, RAC-212]
    branch: agent-2-sec-websocket

  - name: controller-ops
    crew: quick-fix
    issues: [RAC-187, RAC-188, RAC-189]
    branch: agent-3-ops

  - name: ui-portal
    crew: feature-development
    issues: [RAC-169]
    branch: agent-4-ui

phases:
  - name: build
    workstreams: [sec-session, sec-websocket, controller-ops, ui-portal]
    parallel: true

  - name: verify
    workstreams: [sec-session]
    depends_on: [build]

gate:
  checks:
    - "go build ./..."
    - "go test -race ./..."
    - "golangci-lint run"
```

### Config Reference

| Field | Description |
|-------|-------------|
| `name` | Sprint identifier (matches filename without extension) |
| `description` | Human-readable sprint description |
| `workstreams[].name` | Unique workstream name |
| `workstreams[].crew` | Agent crew profile to use |
| `workstreams[].issues` | Linear issue IDs assigned to this workstream |
| `workstreams[].branch` | Git branch the agent works on |
| `workstreams[].working_dir` | Working directory (optional) |
| `phases[].name` | Phase name |
| `phases[].workstreams` | Workstreams active in this phase |
| `phases[].parallel` | Run listed workstreams concurrently |
| `phases[].depends_on` | Phases that must complete first |
| `gate.checks` | Shell commands that must pass (build, test, lint) |

### Validate Before Running

```bash
cadre sprint start S02 --dry-run
```

This parses the config, validates workstream/phase references, and prints the execution plan without launching agents.

## 4. Execute the Sprint

### Start

```bash
cadre sprint start S02
```

This:
1. Loads the sprint config
2. Creates a session with a shared SQLite IPC database
3. Spawns one Claude Code instance per workstream
4. Each agent receives its issue list, branch, and MCP tools for IPC
5. Executes phases in dependency order (parallel within a phase if configured)

### Monitor

```bash
cadre sprint status S02
```

Output:

```
Sprint: S02

AGENT                STATUS       PID      TASK/SUMMARY
-----                ------       ---      ------------
sec-session          running      12345    Working on RAC-127
sec-websocket        running      12346    Working on RAC-128
controller-ops       idle         12347    Waiting for assignment
ui-portal            complete     0        Implemented dashboard
```

### Stop

```bash
cadre sprint stop S02
```

Sends SIGTERM to all running agent processes.

## 5. Post-Sprint

### Run Gate Checks

```bash
cadre sprint gate S02
```

Executes all commands listed in `gate.checks` and reports pass/fail.

### Update Linear Issue States

After reviewing the work, update issue states:

```bash
# Move completed issues to Done
cadre linear issues update RAC-127 --state "Done"
cadre linear issues update RAC-209 --state "Done"
cadre linear issues update RAC-210 --state "Done"

# Or use bulk update
cadre linear issues update RAC-127,RAC-209,RAC-210 --state "Done"
```

### Review and Merge Branches

Each workstream produces commits on its own branch. Review and merge as normal:

```bash
# Review agent branches
git log main..agent-1-sec-session --oneline
git diff main...agent-1-sec-session

# Merge (or create PRs)
git checkout main
git merge --squash agent-1-sec-session
git commit -m "feat: session hardening (RAC-127, RAC-209, RAC-210)"
```

## Tips

- **Sprint configs are discoverable**: the loader walks parent directories looking for `.cadre-workspace/sprints/`, so a config in the monorepo root works from any subdirectory.
- **Start small**: a sprint with 1-2 workstreams is easier to monitor than 4+.
- **Gate checks catch regressions**: always define `go build`, `go test`, and lint as gate checks.
- **IPC enables coordination**: agents can query each other via MCP tools (`send_query`, `check_queries`, `send_response`) for cross-workstream questions.
