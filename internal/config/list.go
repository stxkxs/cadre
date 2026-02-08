package config

import (
	"os"
	"path/filepath"
	"strings"
)

// LoadAgentList returns all agent names
func LoadAgentList() ([]string, error) {
	return listYAMLFiles("agents")
}

// LoadTaskList returns all task names
func LoadTaskList() ([]string, error) {
	return listYAMLFiles("tasks")
}

// LoadCrewList returns all crew names
func LoadCrewList() ([]string, error) {
	return listYAMLFiles("crews")
}

// LoadToolList returns all tool names
func LoadToolList() ([]string, error) {
	return listYAMLFiles("tools")
}

func listYAMLFiles(dir string) ([]string, error) {
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
			names = append(names, stripExt(name))
		}
	}

	return names, nil
}

func stripExt(name string) string {
	ext := filepath.Ext(name)
	return name[:len(name)-len(ext)]
}
