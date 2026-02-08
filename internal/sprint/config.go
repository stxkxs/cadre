package sprint

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// SprintConfig defines a sprint execution plan.
type SprintConfig struct {
	Name        string             `yaml:"name"`
	Description string             `yaml:"description"`
	Workstreams []WorkstreamConfig `yaml:"workstreams"`
	Phases      []PhaseConfig      `yaml:"phases"`
	Gate        GateConfig         `yaml:"gate"`
}

// WorkstreamConfig defines a single workstream (agent assignment) in a sprint.
type WorkstreamConfig struct {
	Name       string   `yaml:"name"`
	Crew       string   `yaml:"crew"`
	Issues     []string `yaml:"issues"`
	Branch     string   `yaml:"branch"`
	WorkingDir string   `yaml:"working_dir"`
}

// PhaseConfig defines an execution phase within a sprint.
type PhaseConfig struct {
	Name        string   `yaml:"name"`
	Workstreams []string `yaml:"workstreams"`
	Parallel    bool     `yaml:"parallel"`
	DependsOn   []string `yaml:"depends_on"`
}

// GateConfig defines quality gate checks run after all phases.
type GateConfig struct {
	Checks []string `yaml:"checks"`
}

// LoadSprint loads a sprint config by name.
// It searches ./sprints/, .cadre-workspace/sprints/, and walks up parent
// directories looking for .cadre-workspace/sprints/.
func LoadSprint(name string) (*SprintConfig, error) {
	candidates := sprintSearchPaths(name)

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read sprint config: %w", err)
		}

		var cfg SprintConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parse sprint config %s: %w", path, err)
		}

		if err := validateSprint(&cfg); err != nil {
			return nil, fmt.Errorf("validate sprint %s: %w", name, err)
		}

		return &cfg, nil
	}

	return nil, fmt.Errorf("sprint config not found: %s (searched sprints/ and .cadre-workspace/sprints/ up to filesystem root)", name)
}

// ListSprints returns names of available sprint configs.
func ListSprints() ([]string, error) {
	dirs := sprintDirs()
	seen := make(map[string]bool)
	var names []string

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read sprint directory %s: %w", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			ext := filepath.Ext(name)
			if ext != ".yaml" && ext != ".yml" {
				continue
			}
			base := name[:len(name)-len(ext)]
			if base == "template" {
				continue
			}
			if !seen[base] {
				seen[base] = true
				names = append(names, base)
			}
		}
	}

	return names, nil
}

// sprintSearchPaths returns candidate file paths for a sprint config.
// Searches CWD then walks up parents looking for .cadre-workspace.
func sprintSearchPaths(name string) []string {
	file := name + ".yaml"
	paths := []string{
		filepath.Join("sprints", file),
		filepath.Join(".cadre-workspace", "sprints", file),
	}

	// Walk up parent directories looking for .cadre-workspace/sprints/
	cwd, err := os.Getwd()
	if err != nil {
		return paths
	}
	dir := filepath.Dir(cwd) // start one level up (CWD already covered above)
	for {
		candidate := filepath.Join(dir, ".cadre-workspace", "sprints", file)
		paths = append(paths, candidate)

		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}
		dir = parent
	}

	return paths
}

// sprintDirs returns directories to search for sprint configs.
func sprintDirs() []string {
	dirs := []string{
		"sprints",
		filepath.Join(".cadre-workspace", "sprints"),
	}

	cwd, err := os.Getwd()
	if err != nil {
		return dirs
	}
	dir := filepath.Dir(cwd)
	for {
		candidate := filepath.Join(dir, ".cadre-workspace", "sprints")
		dirs = append(dirs, candidate)

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return dirs
}

func validateSprint(cfg *SprintConfig) error {
	if cfg.Name == "" {
		return fmt.Errorf("sprint name is required")
	}
	if len(cfg.Workstreams) == 0 {
		return fmt.Errorf("at least one workstream is required")
	}

	wsNames := make(map[string]bool)
	for _, ws := range cfg.Workstreams {
		if ws.Name == "" {
			return fmt.Errorf("workstream name is required")
		}
		if wsNames[ws.Name] {
			return fmt.Errorf("duplicate workstream name: %s", ws.Name)
		}
		wsNames[ws.Name] = true
		if ws.Crew == "" {
			return fmt.Errorf("workstream %s: crew is required", ws.Name)
		}
	}

	phaseNames := make(map[string]bool)
	for _, ph := range cfg.Phases {
		if ph.Name == "" {
			return fmt.Errorf("phase name is required")
		}
		if phaseNames[ph.Name] {
			return fmt.Errorf("duplicate phase name: %s", ph.Name)
		}
		phaseNames[ph.Name] = true
		for _, ws := range ph.Workstreams {
			if !wsNames[ws] {
				return fmt.Errorf("phase %s references unknown workstream: %s", ph.Name, ws)
			}
		}
		for _, dep := range ph.DependsOn {
			if !phaseNames[dep] {
				return fmt.Errorf("phase %s depends on unknown phase: %s", ph.Name, dep)
			}
		}
	}

	return nil
}
