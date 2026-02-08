# Runbook: Project Setup

## Overview

Initialize and configure a new cadre project for AI agent orchestration.

## When to Use

- Starting a new cadre-powered project
- Setting up cadre in an existing repository
- Configuring API keys and providers

## Prerequisites

- [ ] cadre CLI installed (`make install`)
- [ ] Anthropic API key

## Procedure

### Step 1: Install Cadre CLI

```bash
# Clone and build
git clone https://github.com/cadre-oss/cadre.git
cd cadre
make build

# Install globally
make install
# Or use directly
./build/cadre --help
```

### Step 2: Configure API Key

```bash
# Set environment variable
export ANTHROPIC_API_KEY="sk-ant-xxxx"

# Add to shell profile for persistence
echo 'export ANTHROPIC_API_KEY="sk-ant-xxxx"' >> ~/.bashrc

# Verify
echo $ANTHROPIC_API_KEY | head -c 10
```

### Step 3: Initialize Project

```bash
# Create new project directory
cadre init my-project
cd my-project

# Or initialize in existing directory
cd existing-project
cadre init
```

### Step 4: Verify Project Structure

```bash
# Expected structure
ls -la
# cadre.yaml          - Project configuration
# agents/             - Agent definitions
# tasks/              - Task definitions
# cadres/             - Cadre workflows
# tools/              - Custom tool configs
# .cadre/             - Runtime data (gitignored)

# Validate configuration
cadre config validate
```

### Step 5: Configure Project

```yaml
# cadre.yaml
name: my-project
version: "1.0"

provider:
  name: anthropic
  model: claude-sonnet-4-20250514

defaults:
  timeout: 30m
  max_retries: 3

state:
  driver: sqlite  # or memory
  path: .cadre/state.db

logging:
  level: info
  format: json
```

### Step 6: Create First Agent

```yaml
# agents/developer.yaml
name: developer
role: Software Engineer
goal: Write clean, tested code based on requirements
backstory: |
  You are an experienced software engineer who follows
  best practices and writes maintainable code.
tools:
  - file_read
  - file_write
  - bash
  - grep
memory:
  type: conversation
  max_tokens: 100000
```

### Step 7: Create First Task

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
```

### Step 8: Create First Cadre

```yaml
# cadres/development.yaml
name: development
description: Basic development workflow
agents:
  - developer
process: sequential
tasks:
  - name: implement
    agent: developer
```

### Step 9: Test Setup

```bash
# List components
cadre agent list
cadre task list
cadre tool list

# Test agent interactively
cadre agent chat developer

# Run a task
cadre run --task implement --input requirements="Create a hello world function"

# Run full cadre
cadre run development
```

## Configuration Reference

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `ANTHROPIC_API_KEY` | Anthropic API key | Yes |
| `CADRE_LOG_LEVEL` | Log level (debug, info, warn, error) | No |
| `CADRE_CONFIG` | Path to cadre.yaml | No |

### Provider Options

```yaml
provider:
  name: anthropic
  model: claude-sonnet-4-20250514  # or claude-opus-4-20250514
  api_key: ${ANTHROPIC_API_KEY}    # Or hardcode (not recommended)
  max_tokens: 4096
  temperature: 0.7
```

## Verification

Setup is complete when:

- [ ] `cadre config validate` passes
- [ ] `cadre agent list` shows agents
- [ ] `cadre agent chat` works
- [ ] `cadre run` executes successfully

## Troubleshooting

### API Key Invalid

```bash
# Verify key format
echo $ANTHROPIC_API_KEY | grep "^sk-ant-"

# Test with curl
curl https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -d '{"model":"claude-sonnet-4-20250514","max_tokens":10,"messages":[{"role":"user","content":"Hi"}]}'
```

### Config Validation Fails

```bash
# Check YAML syntax
cat cadre.yaml | yq .

# Validate against schema
cadre config validate --verbose
```

## Related

- [Creating Agents](creating-agents.md) - Agent definition
- [Designing Workflows](designing-workflows.md) - Cadre workflows
