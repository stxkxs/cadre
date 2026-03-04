# Cadre

Visual workflow builder for orchestrating AI agents.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/stxkxs/cadre/actions/workflows/ci.yml/badge.svg)](https://github.com/stxkxs/cadre/actions/workflows/ci.yml)

```
┌─────────┐     ┌───────────┐     ┌───────────┐     ┌──────────┐
│  Input   │────▶│  Agent    │────▶│ Condition  │──┬─▶│  Output  │
│          │     │ (Claude)  │     │ (evaluate) │  │  │ (result) │
└─────────┘     └───────────┘     └───────────┘  │  └──────────┘
                                                  │  ┌──────────┐
                                                  └─▶│  Agent   │
                                                     │ (GPT-4o) │
                                                     └──────────┘
```

## What is Cadre?

Cadre is a graph-based workflow builder for designing and running multi-step AI agent pipelines. Connect nodes visually, configure providers, and execute workflows with live streaming output. Build complex orchestration patterns — parallel branches, conditional routing, loops — without writing glue code.

## Features

- **Visual graph editor** — drag-and-drop nodes, snap-to-grid, minimap, undo/redo
- **4 AI providers** — Anthropic (Claude), OpenAI (GPT-4o), Groq (Llama), Claude Code (CLI)
- **Parallel execution** — branches run concurrently with automatic merge
- **Conditional routing** — evaluate expressions to route between branches
- **Live monitoring** — SSE streaming of node outputs during execution
- **Workspace files** — Claude Code nodes can read/write files in a workspace directory
- **Workflow variables** — key-value pairs injected into execution context
- **Export/import** — share workflows as JSON files
- **Pre-built templates** — starter workflows to get going quickly
- **Per-provider cost estimation** — track token usage and costs per run
- **API key encryption** — AES-256-GCM with per-user derived keys
- **Rate limiting** — sliding window rate limits on API endpoints

## Quick Start

### Prerequisites

- Node.js 20+
- PostgreSQL 16+
- pnpm

### Setup

```bash
# Clone the repository
git clone https://github.com/stxkxs/cadre.git
cd cadre

# Install dependencies
pnpm install

# Configure environment
cp .env.example .env
# Edit .env with your database URL, GitHub OAuth credentials, and encryption secret

# Push database schema
pnpm db:push

# Start development server
pnpm dev
```

Open [http://localhost:3000](http://localhost:3000) to access the app.

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `DATABASE_URL` | PostgreSQL connection string | Yes |
| `AUTH_SECRET` | NextAuth session secret (generate with `openssl rand -base64 32`) | Yes |
| `AUTH_GITHUB_ID` | GitHub OAuth app client ID | Yes |
| `AUTH_GITHUB_SECRET` | GitHub OAuth app client secret | Yes |
| `ENCRYPTION_SECRET` | API key encryption secret (generate with `openssl rand -base64 32`) | Yes |
| `NEXTAUTH_URL` | App URL (default: `http://localhost:3000`) | No |

See [`.env.example`](.env.example) for all available variables.

## Architecture

```
User → Graph Editor (React Flow + Zustand)
         ↓
       Save → API Routes (Next.js App Router)
         ↓
       Execute → Engine
         ├── Graph (topology, validation)
         ├── Scheduler (BFS batching, parallel detection)
         └── Executor (node dispatch, retry, timeout)
               ↓
             Provider Registry
               ├── Anthropic (Claude API)
               ├── OpenAI (Chat Completions)
               ├── Groq (Chat Completions)
               └── Claude Code (CLI subprocess)
```

**Engine pipeline**: The graph is validated (cycles, missing providers), then topologically sorted. The scheduler walks the graph in BFS order, batching nodes whose predecessors are all complete. Parallel branches execute concurrently via `Promise.all`. Each node has configurable timeout and retry with exponential backoff.

**Tech stack**: Next.js 16 (App Router), TypeScript, Tailwind CSS v4, shadcn/ui, React Flow, Zustand, Drizzle ORM, PostgreSQL, NextAuth v5.

## Deployment

### Docker Compose (local)

```bash
docker compose up
```

### Kubernetes (Helm)

```bash
helm install cadre ./helm/cadre \
  --set env.DATABASE_URL="postgresql://..." \
  --set env.AUTH_SECRET="..." \
  --set env.AUTH_GITHUB_ID="..." \
  --set env.AUTH_GITHUB_SECRET="..."
```

See [`helm/cadre/`](helm/cadre/) for the full chart and configurable values.

## Development

```bash
pnpm dev          # Start dev server
pnpm build        # Production build
pnpm lint         # ESLint
pnpm test         # Run tests
pnpm test:watch   # Watch mode
pnpm db:studio    # Drizzle Studio (database GUI)
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, code style, and PR guidelines.

## License

[MIT](LICENSE)
