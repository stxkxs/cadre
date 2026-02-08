# Runbook: Documentation Generation Pattern

## Overview

Automated documentation generation using AI agents.

## When to Use

- Generate API documentation
- Create README files
- Document code changes
- Generate runbooks

## Prerequisites

- [ ] Cadre project set up
- [ ] documenter agent defined

## Procedure

### Step 1: Define Documenter Agent

```yaml
# agents/documenter.yaml
name: documenter
role: Technical Writer
goal: Create clear, comprehensive documentation
backstory: |
  You are a technical writer who creates documentation that
  developers love to read. You:
  - Write clear, concise explanations
  - Include practical examples
  - Use consistent formatting
  - Structure content logically
  - Focus on the reader's needs
tools:
  - file_read
  - file_write
  - grep
memory:
  type: conversation
  max_tokens: 50000
```

### Step 2: Define Documentation Tasks

```yaml
# tasks/generate-readme.yaml
name: generate-readme
description: Generate or update README.md
agent: documenter
inputs:
  - name: project_path
    type: string
    required: true
outputs:
  - name: readme_content
    type: string
timeout: 15m
```

```yaml
# tasks/generate-api-docs.yaml
name: generate-api-docs
description: Generate API documentation from code
agent: documenter
inputs:
  - name: source_files
    type: string[]
    required: true
  - name: output_format
    type: string
    default: "markdown"
outputs:
  - name: documentation
    type: string
timeout: 20m
```

### Step 3: Create Documentation Cadre

```yaml
# cadres/documentation.yaml
name: documentation
description: Generate project documentation
agents:
  - documenter
process: sequential
tasks:
  - name: generate-readme
    agent: documenter
  - name: generate-api-docs
    agent: documenter
```

### Step 4: Run Documentation Generation

```bash
# Generate README
cadre run --task generate-readme --input project_path="."

# Generate API docs
cadre run --task generate-api-docs --input source_files='["pkg/**/*.go"]'

# Full documentation suite
cadre run documentation
```

### Step 5: Review and Commit

```bash
# Review generated docs
git diff docs/

# Commit
git add docs/ README.md
git commit -m "docs: update documentation"
```

### Integration: Automated Docs Update

```yaml
# .github/workflows/docs.yml
name: Update Documentation
on:
  push:
    branches: [main]
    paths:
      - 'pkg/**'
      - 'cmd/**'

jobs:
  docs:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4

    - name: Generate documentation
      run: |
        cadre run documentation

    - name: Commit changes
      run: |
        git config user.name "github-actions[bot]"
        git config user.email "github-actions[bot]@users.noreply.github.com"
        git add docs/ README.md
        git diff --staged --quiet || git commit -m "docs: auto-update documentation"
        git push
```

## Documentation Types

### README Generation

```yaml
inputs:
  project_path: "."
  template: |
    # Project Name

    ## Overview
    [Generated description]

    ## Installation
    [Generated from Makefile/go.mod]

    ## Usage
    [Generated from examples/cmd]

    ## API
    [Generated from pkg/]
```

### API Documentation

```yaml
inputs:
  source_files: ["pkg/**/*.go"]
  output_format: "markdown"
  include:
    - functions
    - types
    - examples
```

### Change Documentation

```yaml
inputs:
  commit_range: "HEAD~5..HEAD"
  output: "CHANGELOG.md"
  format: "keep-a-changelog"
```

## Verification

Documentation is correctly generated when:

- [ ] README is accurate and complete
- [ ] API docs match code
- [ ] Examples are valid
- [ ] Formatting is consistent

## Related

- [Code Review](code-review.md) - Review generated docs
- [Creating Agents](../creating-agents.md) - Customize documenter
