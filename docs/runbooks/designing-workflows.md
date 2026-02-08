# Runbook: Designing Workflows

## Overview

Build multi-agent workflows that coordinate tasks between agents.

## When to Use

- Automating multi-step processes
- Coordinating multiple agents
- Creating reusable workflows

## Prerequisites

- [ ] Agents defined
- [ ] Tasks defined
- [ ] Understanding of process types

## Procedure

### Step 1: Define Tasks

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
  - name: summary
    type: string
timeout: 30m
retry:
  max_attempts: 3
  backoff: exponential
```

```yaml
# tasks/review.yaml
name: review
description: Review code changes for quality and security
agent: reviewer
inputs:
  - name: files
    type: string[]
    required: true
outputs:
  - name: approved
    type: boolean
  - name: feedback
    type: string
timeout: 15m
```

### Step 2: Create Cadre Definition

```yaml
# cadres/development.yaml
name: development
description: Development workflow with code review
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

### Step 3: Define Task Dependencies

```yaml
# DAG-based dependencies
tasks:
  - name: analyze
    agent: analyst

  - name: implement
    agent: developer
    depends_on:
      - analyze

  - name: test
    agent: tester
    depends_on:
      - implement

  - name: review
    agent: reviewer
    depends_on:
      - implement
      - test  # Waits for both
```

### Step 4: Configure Process Type

#### Sequential Process

```yaml
# Tasks run one after another
process: sequential
tasks:
  - name: step1
  - name: step2
  - name: step3
```

#### Parallel Process (Future)

```yaml
# Independent tasks run concurrently
process: parallel
tasks:
  - name: task_a  # Runs in parallel
  - name: task_b  # Runs in parallel
  - name: final
    depends_on:
      - task_a
      - task_b
```

#### Hierarchical Process (Future)

```yaml
# Manager delegates to workers
process: hierarchical
manager: lead
workers:
  - developer
  - reviewer
  - tester
```

### Step 5: Handle Inputs and Outputs

```yaml
# Cadre with input/output
name: feature-development
inputs:
  - name: requirements
    type: string
    required: true
outputs:
  - name: implementation_summary
    type: string
  - name: review_status
    type: string

tasks:
  - name: implement
    inputs:
      requirements: ${cadre.inputs.requirements}

  - name: review
    inputs:
      files: ${implement.outputs.files_changed}
```

### Step 6: Add Error Handling

```yaml
# Task-level retry
tasks:
  - name: implement
    retry:
      max_attempts: 3
      backoff: exponential
      on_failure: skip  # or fail, retry

# Cadre-level error handling
error_handling:
  on_task_failure: continue  # or stop
  notify: slack:#alerts
```

### Step 7: Run Workflow

```bash
# Run default cadre
cadre run

# Run specific cadre
cadre run development

# Run with inputs
cadre run development --input requirements="Add user authentication"

# Dry run (preview)
cadre run development --dry-run
```

### Step 8: Monitor Execution

```bash
# Watch execution
cadre status --watch

# View logs
cadre logs

# Filter by agent
cadre logs --agent developer
```

## Workflow Patterns

### Linear Pipeline

```yaml
tasks:
  - name: analyze
  - name: implement
    depends_on: [analyze]
  - name: test
    depends_on: [implement]
  - name: deploy
    depends_on: [test]
```

### Fan-Out/Fan-In

```yaml
tasks:
  - name: plan

  # Fan out
  - name: frontend
    depends_on: [plan]
  - name: backend
    depends_on: [plan]
  - name: database
    depends_on: [plan]

  # Fan in
  - name: integrate
    depends_on: [frontend, backend, database]
```

### Review Loop

```yaml
tasks:
  - name: implement

  - name: review
    depends_on: [implement]

  - name: revise
    depends_on: [review]
    condition: review.outputs.approved == false
```

## Verification

Workflow is correctly designed when:

- [ ] All task dependencies are valid
- [ ] No circular dependencies
- [ ] `cadre run --dry-run` succeeds
- [ ] Workflow completes successfully

## Related

- [Creating Agents](creating-agents.md) - Agent definitions
- [Tool Integration](tool-integration.md) - Tools for tasks
- [Troubleshooting](troubleshooting.md) - Debug workflows
