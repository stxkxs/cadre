# Hello Example

A simple example demonstrating cadre basics with a greeting workflow.

## Overview

This example creates a two-agent crew:
1. **Greeter**: Creates personalized greetings
2. **Poet**: Transforms greetings into poetry

## Usage

```bash
# Navigate to this directory
cd examples/hello

# Run the crew
cadre run greeting-crew --input name="Alice" --input occasion="birthday"

# Or run individual tasks
cadre task run create-greeting --input name="Bob"

# Interactive chat with an agent
cadre agent chat greeter
```

## Project Structure

```
hello/
├── cadre.yaml             # Project config
├── agents/
│   ├── greeter.yaml       # Greeter agent
│   └── poet.yaml          # Poet agent
├── tasks/
│   ├── create-greeting.yaml
│   └── poetize.yaml
└── crews/
    └── greeting-crew.yaml # Workflow definition
```

## Customization

Try modifying the agents' backstories or adding new tasks to extend the workflow!
