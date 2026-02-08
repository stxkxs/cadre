# Runbook: Code Review Pattern

## Overview

Automated code review workflow using AI agents.

## When to Use

- Automated PR review
- Pre-commit checks
- Code quality gates

## Prerequisites

- [ ] Cadre project set up
- [ ] reviewer agent defined

## Procedure

### Step 1: Define Reviewer Agent

```yaml
# agents/reviewer.yaml
name: reviewer
role: Senior Code Reviewer
goal: Ensure code quality, security, and maintainability
backstory: |
  You are an experienced code reviewer who focuses on:
  - Code correctness and logic errors
  - Security vulnerabilities (OWASP Top 10)
  - Performance issues
  - Code style and maintainability
  - Test coverage gaps

  You provide constructive, specific feedback with examples.
tools:
  - file_read
  - grep
memory:
  type: conversation
  max_tokens: 50000
```

### Step 2: Define Review Task

```yaml
# tasks/code-review.yaml
name: code-review
description: Review code changes for quality and security issues
agent: reviewer
inputs:
  - name: files
    type: string[]
    required: true
  - name: focus
    type: string
    required: false
    default: "all"
outputs:
  - name: issues
    type: object[]
  - name: approved
    type: boolean
  - name: summary
    type: string
timeout: 15m
```

### Step 3: Create Review Cadre

```yaml
# cadres/code-review.yaml
name: code-review
description: Automated code review workflow
agents:
  - reviewer
process: sequential
tasks:
  - name: code-review
    agent: reviewer
```

### Step 4: Run Review

```bash
# Review specific files
cadre run code-review --input files='["main.go", "utils.go"]'

# Review with focus area
cadre run code-review --input files='["auth.go"]' --input focus="security"

# Review all changed files
FILES=$(git diff --name-only HEAD~1)
cadre run code-review --input files="$FILES"
```

### Step 5: Parse Review Output

```bash
# Get structured output
cadre run code-review --output json > review.json

# Example output:
# {
#   "issues": [
#     {"file": "main.go", "line": 42, "severity": "warning", "message": "..."},
#     {"file": "utils.go", "line": 15, "severity": "error", "message": "..."}
#   ],
#   "approved": false,
#   "summary": "Found 2 issues that should be addressed..."
# }
```

### Integration: GitHub Actions

```yaml
# .github/workflows/review.yml
name: AI Code Review
on:
  pull_request:
    types: [opened, synchronize]

jobs:
  review:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
      with:
        fetch-depth: 0

    - name: Get changed files
      id: files
      run: |
        FILES=$(git diff --name-only origin/${{ github.base_ref }}...HEAD | jq -R -s -c 'split("\n") | map(select(. != ""))')
        echo "files=$FILES" >> $GITHUB_OUTPUT

    - name: Run code review
      run: |
        cadre run code-review --input files='${{ steps.files.outputs.files }}' --output json > review.json

    - name: Post review comments
      uses: actions/github-script@v6
      with:
        script: |
          const review = require('./review.json')
          for (const issue of review.issues) {
            await github.rest.pulls.createReviewComment({
              owner: context.repo.owner,
              repo: context.repo.repo,
              pull_number: context.issue.number,
              body: issue.message,
              path: issue.file,
              line: issue.line
            })
          }
```

## Verification

Review workflow is working when:

- [ ] Agent identifies real issues
- [ ] Output is structured correctly
- [ ] Integration posts comments
- [ ] False positives are minimal

## Related

- [Creating Agents](../creating-agents.md) - Customize reviewer
- [Documentation Generation](documentation-generation.md) - Related workflow
