# Runbook: Creating Agents

## Overview

Define custom AI agents with specific roles, goals, and tool access.

## When to Use

- Adding new capabilities to a cadre
- Specialized agent for specific domain
- Customizing agent behavior

## Prerequisites

- [ ] Cadre project initialized
- [ ] Understanding of agent roles

## Procedure

### Step 1: Plan Agent Role

Define the agent's purpose:

- **Role**: What job title/function?
- **Goal**: What should it accomplish?
- **Tools**: What capabilities does it need?
- **Backstory**: What context shapes its behavior?

### Step 2: Create Agent Definition

```yaml
# agents/my-agent.yaml
name: my-agent
role: [Job Title]
goal: [Primary objective in one sentence]
backstory: |
  [2-3 sentences of context that shapes behavior]

  Include relevant experience, preferences, or constraints.
tools:
  - file_read
  - file_write
  - bash
memory:
  type: conversation
  max_tokens: 100000
```

### Step 3: Agent Examples

#### Developer Agent

```yaml
# agents/developer.yaml
name: developer
role: Senior Software Engineer
goal: Write clean, tested, production-ready code
backstory: |
  You are a senior engineer with 10 years of experience in Go and Python.
  You follow SOLID principles, write comprehensive tests, and always
  consider edge cases. You prefer simple solutions over clever ones.
tools:
  - file_read
  - file_write
  - bash
  - grep
memory:
  type: conversation
  max_tokens: 100000
```

#### Reviewer Agent

```yaml
# agents/reviewer.yaml
name: reviewer
role: Code Reviewer
goal: Ensure code quality, security, and maintainability
backstory: |
  You are a meticulous code reviewer focused on finding bugs,
  security issues, and maintainability problems. You provide
  constructive feedback with specific suggestions.
tools:
  - file_read
  - grep
memory:
  type: conversation
  max_tokens: 50000
```

#### Documentation Agent

```yaml
# agents/documenter.yaml
name: documenter
role: Technical Writer
goal: Create clear, comprehensive documentation
backstory: |
  You are a technical writer who creates documentation that
  developers love. You write clear examples, explain complex
  concepts simply, and maintain consistent style.
tools:
  - file_read
  - file_write
  - grep
memory:
  type: conversation
  max_tokens: 50000
```

#### DevOps Agent

```yaml
# agents/devops.yaml
name: devops
role: DevOps Engineer
goal: Automate infrastructure and deployment
backstory: |
  You are a DevOps engineer expert in Kubernetes, AWS, and CI/CD.
  You write secure, maintainable infrastructure code and follow
  GitOps principles.
tools:
  - file_read
  - file_write
  - bash
  - grep
memory:
  type: conversation
  max_tokens: 100000
```

### Step 4: Configure Memory

```yaml
memory:
  type: conversation       # Keep conversation history
  max_tokens: 100000      # Token limit for memory
  # Future options:
  # type: long_term       # Persistent memory
  # type: shared          # Shared across agents
```

### Step 5: Assign Tools

Available built-in tools:

| Tool | Description | Use Case |
|------|-------------|----------|
| `file_read` | Read file contents | Code analysis |
| `file_write` | Write/modify files | Code generation |
| `bash` | Execute commands | Build, test, deploy |
| `grep` | Search patterns | Code search |

```yaml
tools:
  - file_read    # Always useful
  - file_write   # For agents that modify code
  - bash         # For agents that run commands
  - grep         # For code search
```

### Step 6: Test Agent

```bash
# Validate agent definition
cadre config validate

# List agents
cadre agent list

# Test interactively
cadre agent chat my-agent

# Test with specific prompt
cadre agent test my-agent --prompt "Explain your role and capabilities"
```

### Step 7: Iterate on Backstory

Backstory tips:

- Be specific about expertise
- Include preferred approaches
- Mention constraints or rules
- Keep it focused (2-5 sentences)

```yaml
# Good backstory
backstory: |
  You are a Go developer who follows the standard library style.
  You prefer table-driven tests and avoid external dependencies
  when the standard library suffices. You always handle errors
  explicitly and never use panic for control flow.

# Too vague
backstory: |
  You are a good programmer.
```

## Agent Patterns

### Specialized Expert

```yaml
name: security-expert
role: Security Engineer
goal: Identify and fix security vulnerabilities
backstory: |
  You are a security engineer with expertise in OWASP Top 10,
  secure coding practices, and threat modeling. You think like
  an attacker to find vulnerabilities.
```

### Manager Agent

```yaml
name: manager
role: Technical Lead
goal: Coordinate work and ensure quality
backstory: |
  You are a technical lead who breaks down complex tasks,
  coordinates team efforts, and ensures deliverables meet
  quality standards.
```

### Specialist + Generalist

```yaml
# Specialist - deep expertise
name: database-expert
role: Database Administrator
goal: Optimize database performance and reliability

# Generalist - broad coverage
name: fullstack
role: Full Stack Developer
goal: Build complete features across the stack
```

## Verification

Agent is correctly defined when:

- [ ] `cadre config validate` passes
- [ ] Agent appears in `cadre agent list`
- [ ] `cadre agent chat` responds appropriately
- [ ] Agent uses assigned tools correctly

## Related

- [Project Setup](project-setup.md) - Initial setup
- [Designing Workflows](designing-workflows.md) - Using agents in cadres
- [Tool Integration](tool-integration.md) - Custom tools
