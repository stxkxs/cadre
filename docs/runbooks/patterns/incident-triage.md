# Runbook: Incident Triage Pattern

## Overview

Automated incident triage and initial response using AI agents.

## When to Use

- On-call alert investigation
- Initial incident assessment
- Automated remediation suggestions
- Post-incident analysis

## Prerequisites

- [ ] Cadre project set up
- [ ] Access to monitoring data
- [ ] Incident response agents defined

## Procedure

### Step 1: Define Incident Responder Agent

```yaml
# agents/incident-responder.yaml
name: incident-responder
role: Site Reliability Engineer
goal: Quickly diagnose and triage production incidents
backstory: |
  You are an SRE with deep knowledge of distributed systems.
  When incidents occur, you:
  - Assess severity and impact
  - Identify likely root causes
  - Suggest immediate remediation steps
  - Document findings clearly

  You prioritize user impact and work systematically.
tools:
  - file_read
  - bash
  - grep
memory:
  type: conversation
  max_tokens: 100000
```

### Step 2: Define Triage Task

```yaml
# tasks/incident-triage.yaml
name: incident-triage
description: Investigate and triage a production incident
agent: incident-responder
inputs:
  - name: alert
    type: string
    required: true
  - name: service
    type: string
    required: false
  - name: namespace
    type: string
    default: "production"
outputs:
  - name: severity
    type: string
  - name: likely_cause
    type: string
  - name: recommended_actions
    type: string[]
  - name: investigation_summary
    type: string
timeout: 10m
```

### Step 3: Create Incident Cadre

```yaml
# cadres/incident-response.yaml
name: incident-response
description: Automated incident triage and initial response
agents:
  - incident-responder
process: sequential
tasks:
  - name: incident-triage
    agent: incident-responder
```

### Step 4: Run Incident Triage

```bash
# Triage from alert
cadre run incident-response \
  --input alert="High CPU usage on api-server pods" \
  --input service="api-server" \
  --input namespace="production"

# Get structured output
cadre run incident-response --output json > triage.json
```

### Step 5: Example Investigation Commands

The agent might run commands like:

```bash
# Check pod status
kubectl get pods -n production -l app=api-server

# Check recent events
kubectl get events -n production --sort-by='.lastTimestamp' | tail -20

# Check logs for errors
kubectl logs -n production -l app=api-server --since=5m | grep -i error

# Check resource usage
kubectl top pods -n production -l app=api-server

# Check recent deployments
kubectl rollout history deployment/api-server -n production
```

### Integration: PagerDuty/Opsgenie

```yaml
# .github/workflows/incident-triage.yml
name: Incident Triage
on:
  repository_dispatch:
    types: [pagerduty-alert]

jobs:
  triage:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4

    - name: Configure kubectl
      uses: azure/k8s-set-context@v3
      with:
        kubeconfig: ${{ secrets.KUBECONFIG }}

    - name: Run triage
      run: |
        cadre run incident-response \
          --input alert="${{ github.event.client_payload.alert }}" \
          --input service="${{ github.event.client_payload.service }}" \
          --output json > triage.json

    - name: Update incident
      run: |
        SUMMARY=$(jq -r '.investigation_summary' triage.json)
        ACTIONS=$(jq -r '.recommended_actions | join("\n")' triage.json)
        # Post to incident channel/ticket
```

### Escalation Workflow

```yaml
# cadres/incident-escalation.yaml
name: incident-escalation
description: Full incident response with escalation
agents:
  - incident-responder
  - senior-engineer
process: sequential
tasks:
  - name: initial-triage
    agent: incident-responder

  - name: deep-investigation
    agent: senior-engineer
    depends_on: [initial-triage]
    condition: initial-triage.outputs.severity in ["critical", "high"]
```

## Severity Classification

```yaml
# Agent classifies based on:
severities:
  critical:
    - Complete service outage
    - Data loss
    - Security breach
  high:
    - Degraded performance affecting many users
    - Partial outage
    - Error rate > 5%
  medium:
    - Elevated error rate
    - Single component degraded
    - Non-critical feature unavailable
  low:
    - Minor issues
    - Cosmetic problems
    - Single user reports
```

## Verification

Incident triage is working when:

- [ ] Severity is correctly assessed
- [ ] Root cause is plausible
- [ ] Recommended actions are actionable
- [ ] Summary is clear and complete

## Related

- [Creating Agents](../creating-agents.md) - Customize responder
- [Tool Integration](../tool-integration.md) - Add monitoring tools
