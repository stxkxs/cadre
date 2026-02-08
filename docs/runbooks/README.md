# Runbooks

Operational procedures for the cadre AI agent orchestration framework.

## Index

### Priority 0 (Critical)

| Runbook | Description |
|---------|-------------|
| [project-setup.md](project-setup.md) | Initialize and configure cadre projects |
| [creating-agents.md](creating-agents.md) | Define custom agents |

### Priority 1 (Important)

| Runbook | Description |
|---------|-------------|
| [designing-workflows.md](designing-workflows.md) | Build multi-agent workflows |
| [tool-integration.md](tool-integration.md) | Configure tools for agents |
| [troubleshooting.md](troubleshooting.md) | Debug cadre execution issues |

### Patterns

| Runbook | Description |
|---------|-------------|
| [patterns/code-review.md](patterns/code-review.md) | Automated code review workflow |
| [patterns/documentation-generation.md](patterns/documentation-generation.md) | Generate documentation |
| [patterns/incident-triage.md](patterns/incident-triage.md) | Incident response automation |

## Quick Reference

```bash
# Initialize project
cadre init my-project

# Run a cadre
cadre run development

# Run single task
cadre run --task implement --input requirements="Add login"

# Interactive agent chat
cadre agent chat developer

# Check status
cadre status
```

## Template

All runbooks follow this consistent format:

```markdown
# Runbook: [Title]

## Overview
Brief description.

## When to Use
- Scenario 1
- Scenario 2

## Prerequisites
- [ ] Required items

## Procedure
### Step 1: [Action]
...

## Verification
How to confirm success.

## Related
- Links to related docs
```
