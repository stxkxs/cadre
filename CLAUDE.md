# cadre

ai agent orchestration framework for automating developer and product management workflows.

## production readiness

**status:** pre-production (phase 11 — web UI complete)

### what's working

- [x] cli skeleton with cobra
- [x] yaml configuration system with validation
- [x] agent runtime with claude api integration
- [x] built-in tools (file_read, file_write, bash, grep)
- [x] sequential task execution with retries
- [x] dag-based task dependency resolution
- [x] state management with sqlite/memory backends
- [x] checkpoint and resume capability
- [x] quick mode (single-agent fast execution)
- [x] mcp server with sqlite ipc store
- [x] sprint orchestration (multi-agent sessions via claude code)
- [x] inter-agent communication (json-rpc queries/responses)
- [x] parallel task execution with worker pool (fail-fast, complete-running, continue-all)
- [x] event system with lifecycle hooks (shell, webhook, log, pause)
- [x] test foundation with mock provider injection (dag, state, config, runtime tests)
- [x] structured output parsing (JSON block extraction from LLM responses)
- [x] provider retry wrapper (exponential backoff for 429/500/502/503/529)
- [x] panic recovery in goroutines (parallel workers, event bus, sprint sessions)
- [x] recursion safety in ExecuteWithToolResults (depth-bounded)
- [x] hierarchical process (manager agent delegates via tool calls)
- [x] persistent long-term memory (SQLite-backed, write-through, bootstrap on startup)
- [x] shared memory across agents (crew-scoped namespace, multi-agent read/write)
- [x] config validation hardening (crew agent refs, dependency cycles, timeout formats)
- [x] correlation IDs and trace context (context propagation, structured log enrichment)
- [x] metrics export (JSONL file exporter, flush after task/crew completion)
- [x] typed errors with codes and suggestions (CadreError, errors.As support)
- [x] dynamic tool loading (exec shell commands, HTTP endpoints from YAML config)
- [x] integration test framework (TestHarness, event capture, mock provider)
- [x] streaming execution (SSE parsing fix, StreamExecute with callbacks)
- [x] `cadre doctor` (environment validation)
- [x] web UI (`cadre serve` — React dashboard, agent chat, visual pipeline composer, run monitoring)
- [x] `cadre version` (build info, go version, OS/arch, git commit)
- [x] shell completions (bash/zsh/fish/powershell)

### remaining items

| priority | item | status | notes |
|----------|------|--------|-------|
| P0 | real-world workflow testing | todo | validate with actual claude api |

## structure

```
cadre/
├── cmd/cadre/             # cli entrypoint
├── internal/
│   ├── cli/              # cobra commands (sprint, quick, agent, task, doctor, version, completion)
│   ├── config/           # yaml config loading & validation (crew refs, cycle detection, timeout)
│   ├── agent/            # agent abstraction, runtime, memory (long_term, shared, SQLite persistence)
│   ├── task/             # task definition, executor, dag (thread-safe Task struct)
│   ├── tool/             # tool interface, registry, dynamic loading (exec, http, loader)
│   │   └── builtin/      # file, bash, grep tools
│   ├── crew/             # orchestrator, parallel coordinator, hierarchical coordinator, shared memory
│   ├── errors/           # typed errors with codes and suggestions (CadreError)
│   ├── event/            # event bus, hooks (shell, webhook, log, pause)
│   ├── provider/         # llm provider interface
│   │   └── anthropic/    # claude api client (SSE streaming)
│   ├── state/            # state management & checkpoints
│   ├── testutil/         # mock provider, mock tool, test harness
│   ├── quick/            # quick mode executor
│   ├── mcp/              # mcp server (sqlite ipc store, json-rpc)
│   ├── sprint/           # sprint config loader, session lifecycle, launcher
│   └── telemetry/        # logging, metrics, trace context, JSONL exporter
├── test/integration/      # integration tests (crew workflows, memory persistence)
├── pkg/cadre/            # public api for library use
├── web/                  # react frontend (vite + react 19 + typescript + tailwind)
│   ├── src/
│   │   ├── api/          # fetch wrappers, sse helpers
│   │   ├── components/   # ui primitives, layout, agents, composer, runs
│   │   ├── hooks/        # tanstack query hooks, sse hooks, chat hook
│   │   ├── pages/        # dashboard, agents, tasks, crews, composer, runs
│   │   ├── types/        # typescript interfaces
│   │   └── styles/       # oklch theme (dark/light + color accents)
│   └── vite.config.ts    # builds to internal/server/dist/ (go:embed)
├── internal/server/      # http server, sse broker, chat manager, handlers
└── examples/             # example projects
```

## commands

```bash
# build
make build

# run tests
make test

# run tests with coverage
make test-coverage

# lint
make lint

# install locally
make install

# initialize new project
./build/cadre init my-project

# run a crew
./build/cadre run development

# run single task
./build/cadre run --task implement --input requirements="add feature"

# interactive agent chat
./build/cadre agent chat developer

# validate configs
./build/cadre config validate

# list agents/tasks/tools
./build/cadre agent list
./build/cadre task list
./build/cadre tool list

# sprint commands
./build/cadre sprint list
./build/cadre sprint start S02
./build/cadre sprint start S02 --dry-run
./build/cadre sprint status S02
./build/cadre sprint stop S02
./build/cadre sprint gate S02

# quick mode
./build/cadre quick fix "login returns 500 error"
./build/cadre quick review ./internal/api/
./build/cadre quick test ./pkg/controller/
./build/cadre quick doc ./internal/webhook/
./build/cadre quick explain ./cmd/main.go

# diagnostics
./build/cadre doctor
./build/cadre version

# shell completions
./build/cadre completion bash
./build/cadre completion zsh
./build/cadre completion fish

# web ui
./build/cadre serve              # start web UI on :8080
./build/cadre serve -p 3000      # custom port
make web-build                    # build frontend
make web-dev                      # vite dev server on :5173

# integration tests
make test-integration
```

## sprint configuration

sprint configs live in `sprints/` or `.cadre-workspace/sprints/`. the loader walks parent directories, so monorepo-root configs are discoverable from subdirectories.

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

## configuration patterns

### cadre.yaml (project config)

```yaml
name: my-project
version: "1.0"

provider:
  name: anthropic
  model: claude-sonnet-4-20250514

defaults:
  timeout: 30m
  max_retries: 3

state:
  driver: sqlite
  path: .cadre/state.db
```

### agent definition

```yaml
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

### task definition

```yaml
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
dependencies: []
timeout: 30m
retry:
  max_attempts: 3
  backoff: exponential
```

### crew definition (sequential)

```yaml
name: development
agents:
  - developer
  - reviewer
process: sequential
tasks:
  - name: implement
    agent: developer
  - name: review
    agent: reviewer
    depends_on:
      - implement
```

### crew definition (parallel)

```yaml
name: parallel-build
process: parallel
concurrency: 4              # max concurrent tasks (0 = NumCPU)
error_strategy: fail-fast   # fail-fast | complete-running | continue-all
agents:
  - builder
  - tester
tasks:
  - name: build-api
    agent: builder
  - name: build-ui
    agent: builder
  - name: integration-test
    agent: tester
    depends_on: [build-api, build-ui]
```

### crew definition (hierarchical)

```yaml
name: managed-development
process: hierarchical
manager: tech-lead               # manager agent that delegates tasks
agents:
  - tech-lead
  - developer
  - reviewer
tasks:
  - name: implement
    agent: developer             # suggested agent (manager may override)
  - name: review
    agent: reviewer
    depends_on: [implement]
```

### hooks configuration

```yaml
# in cadre.yaml
hooks:
  enabled: true
  hooks:
    - name: notify-completion
      type: webhook
      events: [crew.completed, crew.failed]
      url: https://hooks.slack.com/services/...
      blocking: false
    - name: approve-deploy
      type: pause
      events: [task.started]
      message: "Review before proceeding?"
      blocking: true
    - name: audit-log
      type: shell
      events: [task.completed]
      command: "echo $CADRE_EVENT_JSON >> .cadre/audit.jsonl"
      blocking: false
```

## code style

- follow standard go conventions
- use `golangci-lint` for linting
- cobra for cli commands
- viper for configuration
- structured logging with levels

## testing patterns

```go
// table-driven tests preferred
func TestTaskExecution(t *testing.T) {
    tests := []struct {
        name    string
        input   map[string]interface{}
        want    map[string]interface{}
        wantErr bool
    }{
        // test cases
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

### mock provider usage

```go
import "github.com/cadre-oss/cadre/internal/testutil"

// inject mock provider into agent runtime
mock := &testutil.MockProvider{
    Responses: []*provider.Response{
        {Content: "hello", StopReason: "end_turn"},
    },
}
runtime, _ := agent.NewRuntimeWithProvider(cfg, agentCfg, mock, logger)
result, _ := runtime.Execute(ctx, "prompt")
// mock.CallCount() == 1
```

### test packages

| package | tests | coverage |
|---------|-------|----------|
| `task/dag_test.go` | cycle detection, topo sort, GetReady, IsComplete, Reset | dag logic |
| `state/manager_test.go` | StartRun, CompleteRun, FailRun, concurrent access | state management |
| `config/loader_test.go` | load, defaults, env interpolation, validation | config layer |
| `config/validate_test.go` | circular deps, missing agent refs, bad timeouts, exec/http tool validation | config validation |
| `agent/runtime_test.go` | direct response, tool loops, max iterations, cancellation | agent execution |
| `agent/memory_test.go` | persistence, search, truncation, concurrent access, cross-instance | memory system |
| `crew/parallel_test.go` | concurrency, dependencies, error strategies, output propagation | parallel coordinator |
| `crew/hierarchical_test.go` | delegation, dependency order, failure, reassignment, output propagation | hierarchical coordinator |
| `task/executor_test.go` | JSON block extraction, fallback, malformed JSON, multiple blocks, prompt building | output parsing |
| `provider/retry_test.go` | success, retry on 429/500, no retry on 400/401, max retries, context cancel, stream | retry wrapper |
| `event/bus_test.go` | routing, blocking/async, errors, nil safety, concurrent emit | event bus |
| `event/hook_test.go` | shell, webhook, log, pause hook execution | hook implementations |
| `telemetry/trace_test.go` | trace creation, child spans, context propagation, field extraction | trace context |
| `telemetry/exporter_test.go` | JSONL export, flush with/without exporter | metrics export |
| `errors/errors_test.go` | error codes, wrapping, suggestions, errors.As | typed errors |
| `test/integration/` | sequential workflow, dependency propagation, memory persistence, shared memory | integration tests |

## git workflow

**branch naming:**
```
feat/short-description    # new features
fix/short-description     # bug fixes
refactor/short-description # refactoring
```

**commit messages:** lowercase, conventional format
- `feat: add parallel task execution`
- `fix: resolve checkpoint loading issue`
- `docs: update configuration reference`

## environment variables

| variable | description |
|----------|-------------|
| `ANTHROPIC_API_KEY` | claude api key (required for execution) |
| `CADRE_LOG_LEVEL` | logging level (debug, info, warn, error) |
| `CADRE_CONFIG` | path to config file |

## related resources

- [anthropic api docs](https://docs.anthropic.com/)
- [cobra cli library](https://cobra.dev/)
- [mcp protocol](https://modelcontextprotocol.io/)
