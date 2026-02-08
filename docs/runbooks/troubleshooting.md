# Runbook: Troubleshooting

## Overview

Debug and resolve cadre execution issues.

## When to Use

- Agent errors during execution
- Tool failures
- Workflow not completing
- Unexpected behavior

## Prerequisites

- [ ] Cadre project set up
- [ ] Access to logs

## Procedure

### Enable Debug Logging

```bash
# Set log level
export CADRE_LOG_LEVEL=debug

# Run with verbose output
cadre run --verbose

# View logs
cadre logs --level debug
```

### Check Configuration

```bash
# Validate all configs
cadre config validate

# Show effective config
cadre config show

# Check specific agent
cadre agent show developer

# Check specific task
cadre task show implement
```

### Common Errors

#### API Key Invalid

```bash
# Error: invalid api key
# Check key is set
echo $ANTHROPIC_API_KEY | head -c 10

# Test key
curl https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "content-type: application/json" \
  -d '{"model":"claude-sonnet-4-20250514","max_tokens":10,"messages":[{"role":"user","content":"Hi"}]}'
```

#### Agent Not Found

```bash
# Error: agent 'xyz' not found
# List available agents
cadre agent list

# Check agent file exists
ls -la agents/

# Validate agent definition
cadre config validate --verbose
```

#### Tool Execution Failed

```bash
# Error: tool 'bash' failed
# Check tool is available
cadre tool list

# Test tool directly
cadre tool test bash --input command="echo hello"

# Check permissions
ls -la ./

# Enable debug for tool
CADRE_LOG_LEVEL=debug cadre tool test bash --input command="echo hello"
```

#### Task Timeout

```bash
# Error: task 'implement' timed out
# Check current timeout
grep timeout tasks/implement.yaml

# Increase timeout
# In task definition:
# timeout: 60m

# Or via command line
cadre run --timeout 60m
```

#### Dependency Not Found

```bash
# Error: dependency 'analyze' not found
# Check task exists
cadre task list

# Verify dependency name matches exactly
grep -r "depends_on" tasks/ crews/
```

### Debug Workflow Execution

```bash
# Dry run to check dependencies
cadre run --dry-run

# Run with step-by-step
cadre run --step

# Run specific task only
cadre run --task implement

# Resume from checkpoint
cadre run --resume <checkpoint-id>
```

### Check State and Checkpoints

```bash
# List checkpoints
ls -la .cadre/checkpoints/

# View checkpoint
cadre status --checkpoint <id>

# Clear state and restart
rm -rf .cadre/state.db
cadre run
```

### Analyze Agent Behavior

```bash
# Interactive debugging
cadre agent chat developer

# Check what agent sees
cadre agent context developer

# View agent memory
cadre agent memory developer
```

### Network Issues

```bash
# Error: connection timeout
# Check API connectivity
curl -v https://api.anthropic.com

# Check proxy settings
echo $HTTP_PROXY $HTTPS_PROXY

# Increase timeout
export ANTHROPIC_TIMEOUT=120
```

### Memory Issues

```bash
# Error: memory limit exceeded
# Check memory settings
grep memory agents/*.yaml

# Reduce memory
# In agent definition:
# memory:
#   max_tokens: 50000

# Clear conversation history
cadre agent clear developer
```

## Error Reference

| Error | Cause | Fix |
|-------|-------|-----|
| `invalid api key` | Wrong/missing key | Set ANTHROPIC_API_KEY |
| `agent not found` | Missing definition | Create agent YAML |
| `tool failed` | Tool error | Check tool permissions |
| `timeout` | Task too long | Increase timeout |
| `dependency not found` | Wrong dep name | Fix depends_on |
| `circular dependency` | Dep loop | Redesign workflow |

## Get Diagnostic Info

```bash
# Collect diagnostics
cadre doctor

# Output includes:
# - Version info
# - Config validation
# - API connectivity
# - Tool availability
# - Recent errors
```

## Verification

Issue is resolved when:

- [ ] `cadre config validate` passes
- [ ] Workflow completes successfully
- [ ] No errors in logs
- [ ] Expected output produced

## Related

- [Project Setup](project-setup.md) - Configuration
- [Creating Agents](creating-agents.md) - Agent issues
- [Tool Integration](tool-integration.md) - Tool problems
