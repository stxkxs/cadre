# cadre

> AI agent orchestration for developer and product management workflows.

**cadre** is a CLI framework for orchestrating AI agents via Claude Code. It supports multi-agent sprint sessions with inter-agent communication (IPC) over a shared SQLite database, quick single-agent tasks, and YAML-driven crew/task/agent definitions.

## Features

- **Sprint Orchestration**: Launch multi-agent sessions from YAML sprint configs with phased execution
- **Inter-Agent Communication**: SQLite-backed IPC with JSON-RPC MCP server for agent queries and status updates
- **Quick Mode**: Single-agent execution for common tasks (fix, review, test, doc, explain)
- **Agent/Task/Crew Definitions**: YAML-based configuration with validation and env var interpolation
- **Built-in Tools**: File operations, bash execution, grep search
- **MCP Support**: Integrate any MCP-compatible tool server
- **Checkpointing**: Resume interrupted workflows from where they left off

## Quick Start

### Installation

```bash
git clone https://github.com/stxkxs/cadre.git
cd cadre
make build
./build/cadre --help
```

### Initialize a Project

```bash
# Create a new cadre project
cadre init my-project
cd my-project

# Or initialize in current directory
cadre init
```

### Configure API Keys

```bash
export ANTHROPIC_API_KEY=your-api-key
```

Or add to `cadre.yaml`:

```yaml
provider:
  name: anthropic
  model: claude-sonnet-4-20250514
  api_key: ${env.ANTHROPIC_API_KEY}
```

## CLI Commands

### Sprint Commands

Sprint commands manage multi-agent execution sessions. Sprint configs define workstreams (each mapped to a Claude Code agent), execution phases, and gate checks.

```bash
cadre sprint list                    # List available sprint configurations
cadre sprint start <name>            # Start a sprint session
cadre sprint start <name> --dry-run  # Validate config without launching agents
cadre sprint status <name>           # Show agent statuses for a running sprint
cadre sprint stop <name>             # Stop a running sprint session
cadre sprint gate <name>             # Run gate checks (build, test, lint)
```

### Quick Mode

Quick mode provides fast, single-agent execution for common tasks with reduced overhead.

```bash
cadre quick fix "login returns 500 error"     # Fix a bug
cadre quick review ./internal/api/            # Code review
cadre quick test ./pkg/controller/            # Write tests
cadre quick doc ./internal/webhook/           # Generate docs
cadre quick explain ./cmd/main.go             # Explain code
```

Options: `--json` (JSON output), `--dry-run` (show what would be executed).

### Project & Workflow Commands

```bash
# Project management
cadre init [project-name]     # Initialize new project
cadre config show             # Show configuration
cadre config validate         # Validate all configs

# Running workflows
cadre run                     # Run default crew
cadre run <crew-name>         # Run specific crew
cadre run --task <task>       # Run single task
cadre run --resume <id>       # Resume from checkpoint

# Agent management
cadre agent list              # List agents
cadre agent chat <name>       # Interactive chat

# Task management
cadre task list               # List tasks
cadre task run <name>         # Run task

# Tool management
cadre tool list               # List tools

# Monitoring
cadre status                  # Show execution status
cadre logs                    # View logs
cadre logs --agent <name>     # Filter by agent
```

## Project Structure

```
cadre/
├── cmd/cadre/             # CLI entrypoint
├── internal/
│   ├── cli/              # Cobra commands (sprint, quick, agent, task, etc.)
│   ├── config/           # YAML config loading & validation
│   ├── agent/            # Agent abstraction & runtime
│   ├── task/             # Task definition, executor, DAG
│   ├── tool/             # Tool interface & registry
│   │   └── builtin/      # file, bash, grep tools
│   ├── crew/             # Orchestrator & session
│   ├── provider/         # LLM provider interface
│   │   └── anthropic/    # Claude API client
│   ├── state/            # State management & checkpoints
│   ├── quick/            # Quick mode executor
│   ├── mcp/              # MCP server (SQLite IPC store, JSON-RPC)
│   ├── sprint/           # Sprint config loader, session lifecycle, launcher
│   └── telemetry/        # Logging & metrics
├── pkg/cadre/             # Public API for library use
└── examples/             # Example projects
```

## Sprint Configuration

Sprint configs live in `sprints/` or `.cadre-workspace/sprints/`. Each config defines workstreams (agent assignments), execution phases, and quality gate checks.

```yaml
name: S02
description: "Security & Operator Registration"

workstreams:
  - name: sec-session
    crew: security-hardening
    issues: [SEC-127, SEC-209]
    branch: agent-1-sec-session
  - name: ui-portal
    crew: feature-development
    issues: [UI-169]
    branch: agent-4-ui

phases:
  - name: build
    workstreams: [sec-session, ui-portal]
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

The sprint system walks parent directories looking for `.cadre-workspace/sprints/`, so configs placed in a monorepo root are discoverable from any subdirectory.

## Configuration

### Agent Definition

```yaml
# agents/developer.yaml
name: developer
role: Senior Software Engineer
goal: Write clean, tested, production-ready code
backstory: |
  You are a senior engineer with 10 years of experience.
tools:
  - file_read
  - file_write
  - bash
  - grep
memory:
  type: conversation
  max_tokens: 100000
```

### Task Definition

```yaml
# tasks/implement.yaml
name: implement
description: Implement a feature based on requirements
agent: developer
inputs:
  - name: requirements
    type: string
    required: true
outputs:
  - name: files_changed
    type: string[]
timeout: 30m
retry:
  max_attempts: 3
  backoff: exponential
```

### Crew Definition

```yaml
# crews/development.yaml
name: development
agents:
  - developer
  - reviewer
process: sequential  # sequential | parallel | hierarchical
tasks:
  - name: implement
    agent: developer
  - name: review
    agent: reviewer
    depends_on:
      - implement
```

## Built-in Tools

| Tool | Description |
|------|-------------|
| `file_read` | Read file contents with optional line range |
| `file_write` | Write or append to files |
| `bash` | Execute shell commands |
| `grep` | Search for patterns in files |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `CADRE_LOG_LEVEL` | Logging level (debug, info, warn, error) |
| `CADRE_CONFIG` | Path to config file |

## Development

```bash
make build          # Build
make test           # Run tests
make test-coverage  # Run with coverage
make lint           # Lint
make install        # Install locally
```

## License

MIT
