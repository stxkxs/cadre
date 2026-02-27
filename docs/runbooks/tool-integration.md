# Runbook: Tool Integration

## Overview

Configure and integrate tools for agents to use during execution.

## When to Use

- Adding new capabilities to agents
- Integrating external services
- Custom tool development

## Prerequisites

- [ ] Cadre project set up
- [ ] Understanding of tool capabilities

## Procedure

### Step 1: List Available Tools

```bash
# List built-in tools
cadre tool list

# Output:
# file_read   - Read file contents
# file_write  - Write or append to files
# bash        - Execute shell commands
# grep        - Search for patterns in files
```

### Step 2: Use Built-in Tools

#### File Read Tool

```yaml
# In agent definition
tools:
  - file_read

# Agent can then read files:
# "Read the contents of main.go"
```

#### File Write Tool

```yaml
tools:
  - file_write

# Agent can write files:
# "Create a new file utils.go with..."
```

#### Bash Tool

```yaml
tools:
  - bash

# Agent can run commands:
# "Run go test ./..."
```

#### Grep Tool

```yaml
tools:
  - grep

# Agent can search:
# "Find all uses of 'TODO' in the codebase"
```

### Step 3: Configure Tool Permissions

```yaml
# agents/developer.yaml
tools:
  - file_read
  - file_write:
      # Restrict to specific paths
      allowed_paths:
        - ./src/**
        - ./tests/**
      denied_paths:
        - ./.env
        - ./secrets/**
  - bash:
      # Restrict commands
      allowed_commands:
        - go
        - make
        - npm
      denied_commands:
        - rm -rf
        - sudo
```

### Step 4: Add MCP Server Tools (Future)

```yaml
# tools/mcp.yaml
mcp_servers:
  - name: github
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_TOKEN: ${GITHUB_TOKEN}

  - name: filesystem
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem"]
    config:
      allowed_directories:
        - ./src
```

### Step 5: Create Custom Tool

```go
// tools/custom/mytool.go
package custom

import "github.com/stxkxs/cadre/pkg/tool"

type MyTool struct{}

func (t *MyTool) Name() string {
    return "mytool"
}

func (t *MyTool) Description() string {
    return "Does something custom"
}

func (t *MyTool) Schema() tool.Schema {
    return tool.Schema{
        Type: "object",
        Properties: map[string]tool.Property{
            "input": {Type: "string", Description: "Input value"},
        },
        Required: []string{"input"},
    }
}

func (t *MyTool) Execute(params map[string]interface{}) (interface{}, error) {
    input := params["input"].(string)
    // Do something
    return map[string]interface{}{
        "result": "processed: " + input,
    }, nil
}
```

### Step 6: Register Custom Tool

```yaml
# tools/custom.yaml
custom_tools:
  - name: mytool
    path: ./tools/custom/mytool.go

# In agent
tools:
  - file_read
  - mytool
```

### Step 7: Test Tool Usage

```bash
# Test tool directly
cadre tool test file_read --input path=./README.md

# Test tool via agent
cadre agent chat developer
> Read the contents of main.go
```

## Tool Configuration Examples

### Read-Only Agent

```yaml
name: analyst
tools:
  - file_read
  - grep
# No write or execute permissions
```

### Full Access Agent

```yaml
name: developer
tools:
  - file_read
  - file_write
  - bash
  - grep
```

### Restricted Bash

```yaml
name: tester
tools:
  - file_read
  - bash:
      allowed_commands:
        - go test
        - npm test
        - pytest
      timeout: 5m
```

## Verification

Tool integration is working when:

- [ ] `cadre tool list` shows tools
- [ ] `cadre tool test` executes correctly
- [ ] Agent uses tools in chat
- [ ] Permissions are enforced

## Related

- [Creating Agents](creating-agents.md) - Assign tools
- [Designing Workflows](designing-workflows.md) - Use tools in tasks
