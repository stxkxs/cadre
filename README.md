# cadre

Visual workflow builder for orchestrating Claude Code tasks.

Build graph-based pipelines where each node is a Claude Code session. Connect nodes to pass context between steps. Branch, loop, transform, and gate your workflows — all powered by your local Claude Code CLI.

## Node Types

| Node | Purpose |
|------|---------|
| **Claude Code** | Execute a Claude Code session with a system prompt |
| **Condition** | True/false branching via JavaScript expressions |
| **Router** | Multi-way branching with N labeled routes |
| **Transform** | Data shaping with `{{variable}}` interpolation (no LLM) |
| **Gate** | Pause for manual approval before continuing |
| **Loop** | Repeat until a condition is met |
| **Input** | Workflow entry point |
| **Output** | Workflow exit point |

## Prerequisites

- [Node.js](https://nodejs.org/) 20+
- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) CLI installed and authenticated
- PostgreSQL 16+

## Setup

```bash
# Install dependencies
pnpm install

# Start PostgreSQL
docker compose up -d

# Configure environment
cp .env.example .env
# Edit .env — set AUTH_SECRET and AUTH_PASSWORD

# Initialize database
pnpm db:push

# Start dev server
pnpm dev
```

## Stack

- Next.js 16 (App Router) + TypeScript
- React Flow for the graph editor
- Zustand for state management
- Drizzle ORM + PostgreSQL
- Tailwind CSS v4 + shadcn/ui

## License

MIT
