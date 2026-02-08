package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	initTemplate string
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new cadre project",
	Long: `Initialize a new cadre project with the standard directory structure.

Available templates:
  default - Basic project structure
  pm      - Product management focused
  dev     - Developer focused`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVarP(&initTemplate, "template", "t", "default", "project template (default, pm, dev)")
}

func runInit(cmd *cobra.Command, args []string) error {
	projectName := "."
	if len(args) > 0 {
		projectName = args[0]
	}

	// Create project directory if not current directory
	if projectName != "." {
		if err := os.MkdirAll(projectName, 0755); err != nil {
			return fmt.Errorf("failed to create project directory: %w", err)
		}
	}

	// Create directory structure
	dirs := []string{
		"agents",
		"tasks",
		"tools",
		"crews",
		"flows",
		"prompts",
		".cadre/checkpoints",
		".cadre/memory",
		".cadre/logs",
	}

	for _, dir := range dirs {
		path := filepath.Join(projectName, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create cadre.yaml
	if err := createCrewConfig(projectName); err != nil {
		return err
	}

	// Create template-specific files
	switch initTemplate {
	case "pm":
		if err := createPMTemplate(projectName); err != nil {
			return err
		}
	case "dev":
		if err := createDevTemplate(projectName); err != nil {
			return err
		}
	default:
		if err := createDefaultTemplate(projectName); err != nil {
			return err
		}
	}

	// Create .gitignore
	if err := createGitignore(projectName); err != nil {
		return err
	}

	fmt.Printf("Initialized cadre project in %s\n", projectName)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Configure your API keys in cadre.yaml or environment")
	fmt.Println("  2. Customize agents in agents/")
	fmt.Println("  3. Run 'cadre run' to start your crew")

	return nil
}

func createCrewConfig(projectDir string) error {
	content := `# cadre.yaml - Project configuration
name: my-project
version: "1.0"

# Provider configuration
provider:
  name: anthropic
  model: claude-sonnet-4-20250514
  # api_key: ${ANTHROPIC_API_KEY}

# Default settings
defaults:
  timeout: 30m
  max_retries: 3

# Logging
logging:
  level: info
  format: text  # text | json

# State storage
state:
  driver: sqlite
  path: .cadre/state.db
`
	return os.WriteFile(filepath.Join(projectDir, "cadre.yaml"), []byte(content), 0644)
}

func createDefaultTemplate(projectDir string) error {
	// Developer agent
	developerAgent := `name: developer
role: Senior Software Engineer
goal: Write clean, tested, production-ready code
backstory: |
  You are a senior engineer with 10 years of experience.
  You write idiomatic code, follow best practices, and always include tests.
tools:
  - file_read
  - file_write
  - bash
  - grep
memory:
  type: conversation
  max_tokens: 100000
`
	if err := os.WriteFile(filepath.Join(projectDir, "agents", "developer.yaml"), []byte(developerAgent), 0644); err != nil {
		return err
	}

	// Reviewer agent
	reviewerAgent := `name: reviewer
role: Code Reviewer
goal: Ensure code quality, security, and maintainability
backstory: |
  You are a meticulous code reviewer who catches bugs, security issues,
  and suggests improvements while being constructive and helpful.
tools:
  - file_read
  - grep
memory:
  type: conversation
  max_tokens: 50000
`
	if err := os.WriteFile(filepath.Join(projectDir, "agents", "reviewer.yaml"), []byte(reviewerAgent), 0644); err != nil {
		return err
	}

	// Implement task
	implementTask := `name: implement
description: Implement a feature based on requirements
agent: developer
inputs:
  - name: requirements
    type: string
    required: true
  - name: files
    type: string[]
    required: false
outputs:
  - name: files_changed
    type: string[]
  - name: summary
    type: string
dependencies: []
timeout: 30m
retry:
  max_attempts: 3
  backoff: exponential
`
	if err := os.WriteFile(filepath.Join(projectDir, "tasks", "implement.yaml"), []byte(implementTask), 0644); err != nil {
		return err
	}

	// Review task
	reviewTask := `name: review
description: Review code changes for quality and issues
agent: reviewer
inputs:
  - name: files_changed
    type: string[]
    required: true
outputs:
  - name: approved
    type: boolean
  - name: feedback
    type: string
dependencies:
  - implement
timeout: 15m
`
	if err := os.WriteFile(filepath.Join(projectDir, "tasks", "review.yaml"), []byte(reviewTask), 0644); err != nil {
		return err
	}

	// Default crew
	defaultCrew := `name: development
description: Basic development workflow
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
`
	return os.WriteFile(filepath.Join(projectDir, "crews", "development.yaml"), []byte(defaultCrew), 0644)
}

func createPMTemplate(projectDir string) error {
	// PM agent
	pmAgent := `name: pm
role: Product Manager
goal: Define clear requirements and prioritize work effectively
backstory: |
  You are an experienced product manager who excels at breaking down
  complex features into actionable tasks and ensuring alignment with goals.
tools:
  - file_read
  - file_write
memory:
  type: conversation
  max_tokens: 100000
`
	if err := os.WriteFile(filepath.Join(projectDir, "agents", "pm.yaml"), []byte(pmAgent), 0644); err != nil {
		return err
	}

	// Also create developer for complete workflow
	return createDefaultTemplate(projectDir)
}

func createDevTemplate(projectDir string) error {
	// QA agent
	qaAgent := `name: qa
role: QA Engineer
goal: Ensure software quality through comprehensive testing
backstory: |
  You are a thorough QA engineer who designs test cases,
  finds edge cases, and ensures reliability.
tools:
  - file_read
  - bash
  - grep
memory:
  type: conversation
  max_tokens: 50000
`
	if err := os.WriteFile(filepath.Join(projectDir, "agents", "qa.yaml"), []byte(qaAgent), 0644); err != nil {
		return err
	}

	// Also create base developer template
	return createDefaultTemplate(projectDir)
}

func createGitignore(projectDir string) error {
	content := `# cadre
.cadre/checkpoints/
.cadre/memory/
.cadre/logs/
.cadre/state.db

# Secrets
*.env
.env.*

# OS
.DS_Store
Thumbs.db
`
	return os.WriteFile(filepath.Join(projectDir, ".gitignore"), []byte(content), 0644)
}
