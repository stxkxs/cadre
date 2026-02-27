package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// TemplateCategory represents a template category.
type TemplateCategory struct {
	ID    string `yaml:"id" json:"id"`
	Label string `yaml:"label" json:"label"`
	Icon  string `yaml:"icon" json:"icon"`
}

// TemplateMeta holds metadata for a single template entity.
type TemplateMeta struct {
	Category   string `yaml:"category" json:"category"`
	Complexity string `yaml:"complexity" json:"complexity"` // beginner, intermediate, advanced
}

// TemplateIndex is the top-level structure of templates.yaml.
type TemplateIndex struct {
	Categories []TemplateCategory      `yaml:"categories" json:"categories"`
	Agents     map[string]TemplateMeta `yaml:"agents" json:"agents"`
	Tasks      map[string]TemplateMeta `yaml:"tasks" json:"tasks"`
	Crews      map[string]TemplateMeta `yaml:"crews" json:"crews"`
}

// TemplateAgent is an agent config enriched with template metadata.
type TemplateAgent struct {
	AgentConfig `json:",inline"`
	Meta        TemplateMeta `json:"meta"`
}

// TemplateTask is a task config enriched with template metadata.
type TemplateTask struct {
	TaskConfig `json:",inline"`
	Meta       TemplateMeta `json:"meta"`
}

// TemplateCrew is a crew config enriched with template metadata.
type TemplateCrew struct {
	CrewConfig `json:",inline"`
	Meta       TemplateMeta `json:"meta"`
}

// findTemplatesDir locates the templates.yaml file and examples directory.
// It checks the current working directory first, then walks up from the
// source file location to find the project root.
func findTemplatesDir() string {
	// Try current working directory first
	if _, err := os.Stat("templates.yaml"); err == nil {
		return "."
	}

	// Try to find by walking up from the source file location
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		dir := filepath.Dir(filename)
		for i := 0; i < 5; i++ {
			candidate := filepath.Join(dir, "templates.yaml")
			if _, err := os.Stat(candidate); err == nil {
				return dir
			}
			dir = filepath.Dir(dir)
		}
	}

	return "."
}

// LoadTemplateIndex loads and parses the templates.yaml catalog.
func LoadTemplateIndex() (*TemplateIndex, error) {
	root := findTemplatesDir()
	path := filepath.Join(root, "templates.yaml")

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &TemplateIndex{
				Categories: []TemplateCategory{},
				Agents:     map[string]TemplateMeta{},
				Tasks:      map[string]TemplateMeta{},
				Crews:      map[string]TemplateMeta{},
			}, nil
		}
		return nil, fmt.Errorf("failed to read templates.yaml: %w", err)
	}

	var idx TemplateIndex
	if err := yaml.Unmarshal(content, &idx); err != nil {
		return nil, fmt.Errorf("failed to parse templates.yaml: %w", err)
	}

	if idx.Agents == nil {
		idx.Agents = map[string]TemplateMeta{}
	}
	if idx.Tasks == nil {
		idx.Tasks = map[string]TemplateMeta{}
	}
	if idx.Crews == nil {
		idx.Crews = map[string]TemplateMeta{}
	}

	return &idx, nil
}

// findExamplesDir locates the examples/hello directory.
func findExamplesDir() string {
	candidates := []string{
		"examples/hello",
		filepath.Join(findTemplatesDir(), "examples", "hello"),
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c
		}
	}
	return "examples/hello"
}

// ListTemplateAgents loads all agent templates from the examples directory.
func ListTemplateAgents(idx *TemplateIndex) ([]TemplateAgent, error) {
	exDir := findExamplesDir()
	agentDir := filepath.Join(exDir, "agents")

	names, err := listYAMLFilesInDir(agentDir)
	if err != nil {
		return nil, err
	}

	var result []TemplateAgent
	for _, name := range names {
		content, err := os.ReadFile(filepath.Join(agentDir, name+".yaml"))
		if err != nil {
			continue
		}
		var cfg AgentConfig
		if err := yaml.Unmarshal(content, &cfg); err != nil {
			continue
		}
		meta := idx.Agents[name]
		if meta.Category == "" {
			meta.Category = "general"
		}
		if meta.Complexity == "" {
			meta.Complexity = "beginner"
		}
		result = append(result, TemplateAgent{AgentConfig: cfg, Meta: meta})
	}

	return result, nil
}

// ListTemplateTasks loads all task templates from the examples directory.
func ListTemplateTasks(idx *TemplateIndex) ([]TemplateTask, error) {
	exDir := findExamplesDir()
	taskDir := filepath.Join(exDir, "tasks")

	names, err := listYAMLFilesInDir(taskDir)
	if err != nil {
		return nil, err
	}

	var result []TemplateTask
	for _, name := range names {
		content, err := os.ReadFile(filepath.Join(taskDir, name+".yaml"))
		if err != nil {
			continue
		}
		var cfg TaskConfig
		if err := yaml.Unmarshal(content, &cfg); err != nil {
			continue
		}
		meta := idx.Tasks[name]
		if meta.Category == "" {
			meta.Category = "general"
		}
		if meta.Complexity == "" {
			meta.Complexity = "beginner"
		}
		result = append(result, TemplateTask{TaskConfig: cfg, Meta: meta})
	}

	return result, nil
}

// ListTemplateCrews loads all crew templates from the examples directory.
func ListTemplateCrews(idx *TemplateIndex) ([]TemplateCrew, error) {
	exDir := findExamplesDir()
	crewDir := filepath.Join(exDir, "crews")

	names, err := listYAMLFilesInDir(crewDir)
	if err != nil {
		return nil, err
	}

	var result []TemplateCrew
	for _, name := range names {
		content, err := os.ReadFile(filepath.Join(crewDir, name+".yaml"))
		if err != nil {
			continue
		}
		var cfg CrewConfig
		if err := yaml.Unmarshal(content, &cfg); err != nil {
			continue
		}
		meta := idx.Crews[name]
		if meta.Category == "" {
			meta.Category = "general"
		}
		if meta.Complexity == "" {
			meta.Complexity = "beginner"
		}
		result = append(result, TemplateCrew{CrewConfig: cfg, Meta: meta})
	}

	return result, nil
}

// listYAMLFilesInDir lists YAML files in a specific directory (not the CWD-relative one).
func listYAMLFilesInDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			ext := filepath.Ext(name)
			names = append(names, name[:len(name)-len(ext)])
		}
	}
	return names, nil
}
