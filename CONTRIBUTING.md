# Contributing to Cadre

Thanks for your interest in contributing! Here's how to get started.

## Prerequisites

- Node.js 20+
- PostgreSQL 16+
- pnpm

## Development Setup

1. **Fork and clone** the repository
2. **Install dependencies**: `pnpm install`
3. **Set up environment**: `cp .env.example .env` and fill in values (see [README.md](README.md#2-configure-environment))
4. **Start PostgreSQL**: Docker, local install, or managed service
5. **Push schema**: `pnpm db:push`
6. **Run dev server**: `pnpm dev`

## Making Changes

1. Create a feature branch from `main`
2. Make your changes
3. Run checks before committing:
   ```bash
   pnpm lint        # Zero errors required
   pnpm test        # All tests must pass
   pnpm build       # Must build successfully
   ```
4. Commit with a descriptive message (see conventions below)
5. Open a pull request against `main`

## Code Style

- **TypeScript strict mode** — no `any` types, no implicit returns
- **ESLint** — run `pnpm lint` and fix all issues
- **Formatting** — follows project ESLint config
- **Imports** — use `@/` path alias for `src/` imports

## Commit Messages

Use concise, descriptive commit messages:

```
feat: add workflow duplication
fix: prevent self-loop edge creation
refactor: extract provider streaming logic
test: add scheduler batch ordering tests
docs: update deployment instructions
```

Prefix with the type of change: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`.

## Testing

- Tests live in `__tests__/` directories adjacent to the code they test
- Use Vitest (`pnpm test`) — all tests must pass before merge
- Mock external dependencies (API clients, file system, child processes)
- Test files follow the pattern `*.test.ts`

## Pull Request Guidelines

- Keep PRs focused — one feature or fix per PR
- Include a clear description of what changed and why
- Add tests for new functionality
- Ensure CI passes (lint, typecheck, test)

## Project Structure

```
src/
├── app/              # Next.js App Router pages and API routes
├── components/       # React components (workflow/, layout/, ui/)
├── lib/
│   ├── engine/       # Graph, Scheduler, Executor, Context
│   ├── providers/    # AI provider implementations
│   ├── db/           # Drizzle schema and connection
│   ├── store/        # Zustand stores
│   └── *.ts          # Utilities (crypto, rate-limit, auth, etc.)
└── types/            # Shared TypeScript types
```

## Questions?

Open an issue for bugs, feature requests, or questions.
