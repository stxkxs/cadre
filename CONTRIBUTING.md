# Contributing to Cadre

Thanks for your interest in contributing! Here's how to get started.

## Prerequisites

- Node.js 20+
- pnpm
- Docker (for PostgreSQL)
- Claude Code CLI installed and authenticated

## Setup

```bash
pnpm install
cp .env.example .env
docker compose up -d
pnpm db:push
pnpm dev
```

## Development

```bash
pnpm dev          # Start dev server
pnpm typecheck    # Type check
pnpm lint         # Lint
pnpm test         # Run tests
pnpm build        # Production build
```

## Project Structure

```
src/
  app/              # Next.js App Router pages and API routes
  components/
    layout/         # App shell, sidebar, header
    workflow/        # Graph canvas, node types, config panel, toolbar
    ui/             # shadcn/ui primitives
  lib/
    engine/         # Graph execution engine
    providers/      # Claude Code CLI provider
    store/          # Zustand stores
    db/             # Drizzle ORM schema
  types/            # TypeScript type definitions
```
