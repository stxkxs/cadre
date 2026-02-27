package server

import (
	"net/http"
	"strings"

	"github.com/stxkxs/cadre/internal/config"
)

// --- Templates ---

func (s *Server) handleListTemplates(w http.ResponseWriter, _ *http.Request) {
	idx, err := config.LoadTemplateIndex()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Compute entity counts per category.
	type CategoryInfo struct {
		config.TemplateCategory
		AgentCount int `json:"agent_count"`
		TaskCount  int `json:"task_count"`
		CrewCount  int `json:"crew_count"`
	}

	cats := make([]CategoryInfo, 0, len(idx.Categories))
	for _, cat := range idx.Categories {
		info := CategoryInfo{TemplateCategory: cat}
		for _, meta := range idx.Agents {
			if meta.Category == cat.ID {
				info.AgentCount++
			}
		}
		for _, meta := range idx.Tasks {
			if meta.Category == cat.ID {
				info.TaskCount++
			}
		}
		for _, meta := range idx.Crews {
			if meta.Category == cat.ID {
				info.CrewCount++
			}
		}
		cats = append(cats, info)
	}

	jsonResponse(w, http.StatusOK, cats)
}

func (s *Server) handleListTemplateAgents(w http.ResponseWriter, r *http.Request) {
	idx, err := config.LoadTemplateIndex()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	agents, err := config.ListTemplateAgents(idx)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Apply filters.
	category := r.URL.Query().Get("category")
	query := strings.ToLower(r.URL.Query().Get("q"))

	var filtered []config.TemplateAgent
	for _, a := range agents {
		if category != "" && a.Meta.Category != category {
			continue
		}
		if query != "" && !containsQuery(a.Name, a.Role, a.Goal, query) {
			continue
		}
		filtered = append(filtered, a)
	}

	if filtered == nil {
		filtered = []config.TemplateAgent{}
	}

	jsonResponse(w, http.StatusOK, filtered)
}

func (s *Server) handleGetTemplateAgent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	idx, err := config.LoadTemplateIndex()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	agents, err := config.ListTemplateAgents(idx)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	for _, a := range agents {
		if a.Name == name {
			jsonResponse(w, http.StatusOK, a)
			return
		}
	}

	jsonError(w, http.StatusNotFound, "template agent not found")
}

func (s *Server) handleListTemplateTasks(w http.ResponseWriter, r *http.Request) {
	idx, err := config.LoadTemplateIndex()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tasks, err := config.ListTemplateTasks(idx)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	category := r.URL.Query().Get("category")
	query := strings.ToLower(r.URL.Query().Get("q"))

	var filtered []config.TemplateTask
	for _, t := range tasks {
		if category != "" && t.Meta.Category != category {
			continue
		}
		if query != "" && !containsQuery(t.Name, t.Description, t.Agent, query) {
			continue
		}
		filtered = append(filtered, t)
	}

	if filtered == nil {
		filtered = []config.TemplateTask{}
	}

	jsonResponse(w, http.StatusOK, filtered)
}

func (s *Server) handleGetTemplateTask(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	idx, err := config.LoadTemplateIndex()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tasks, err := config.ListTemplateTasks(idx)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	for _, t := range tasks {
		if t.Name == name {
			jsonResponse(w, http.StatusOK, t)
			return
		}
	}

	jsonError(w, http.StatusNotFound, "template task not found")
}

func (s *Server) handleListTemplateCrews(w http.ResponseWriter, r *http.Request) {
	idx, err := config.LoadTemplateIndex()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	crews, err := config.ListTemplateCrews(idx)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	category := r.URL.Query().Get("category")
	query := strings.ToLower(r.URL.Query().Get("q"))

	var filtered []config.TemplateCrew
	for _, c := range crews {
		if category != "" && c.Meta.Category != category {
			continue
		}
		if query != "" && !containsQuery(c.Name, c.Description, c.Process, query) {
			continue
		}
		filtered = append(filtered, c)
	}

	if filtered == nil {
		filtered = []config.TemplateCrew{}
	}

	jsonResponse(w, http.StatusOK, filtered)
}

func (s *Server) handleGetTemplateCrew(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	idx, err := config.LoadTemplateIndex()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	crews, err := config.ListTemplateCrews(idx)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	for _, c := range crews {
		if c.Name == name {
			jsonResponse(w, http.StatusOK, c)
			return
		}
	}

	jsonError(w, http.StatusNotFound, "template crew not found")
}

func (s *Server) handleImportTemplate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type      string                 `json:"type"` // agent, task, crew
		Name      string                 `json:"name"`
		Overrides map[string]interface{} `json:"overrides"`
	}
	if err := decodeJSON(r, &req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	idx, err := config.LoadTemplateIndex()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	switch req.Type {
	case "agent":
		agents, err := config.ListTemplateAgents(idx)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		for _, a := range agents {
			if a.Name == req.Name {
				if err := writeYAML("agents", a.Name, a.AgentConfig); err != nil {
					jsonError(w, http.StatusInternalServerError, err.Error())
					return
				}
				jsonResponse(w, http.StatusOK, map[string]string{"status": "imported", "name": a.Name})
				return
			}
		}
		jsonError(w, http.StatusNotFound, "template agent not found")

	case "task":
		tasks, err := config.ListTemplateTasks(idx)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		for _, t := range tasks {
			if t.Name == req.Name {
				if err := writeYAML("tasks", t.Name, t.TaskConfig); err != nil {
					jsonError(w, http.StatusInternalServerError, err.Error())
					return
				}
				jsonResponse(w, http.StatusOK, map[string]string{"status": "imported", "name": t.Name})
				return
			}
		}
		jsonError(w, http.StatusNotFound, "template task not found")

	case "crew":
		crews, err := config.ListTemplateCrews(idx)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		for _, c := range crews {
			if c.Name == req.Name {
				// Import the crew itself.
				if err := writeYAML("crews", c.Name, c.CrewConfig); err != nil {
					jsonError(w, http.StatusInternalServerError, err.Error())
					return
				}

				// Transitively import referenced agents that don't already exist.
				existingAgents, _ := config.LoadAgentList()
				existingAgentSet := make(map[string]bool)
				for _, a := range existingAgents {
					existingAgentSet[a] = true
				}

				agents, _ := config.ListTemplateAgents(idx)
				for _, agentName := range c.Agents {
					if existingAgentSet[agentName] {
						continue
					}
					for _, a := range agents {
						if a.Name == agentName {
							writeYAML("agents", a.Name, a.AgentConfig)
							break
						}
					}
				}

				// Transitively import referenced tasks that don't already exist.
				existingTasks, _ := config.LoadTaskList()
				existingTaskSet := make(map[string]bool)
				for _, t := range existingTasks {
					existingTaskSet[t] = true
				}

				tasks, _ := config.ListTemplateTasks(idx)
				for _, crewTask := range c.Tasks {
					if existingTaskSet[crewTask.Name] {
						continue
					}
					for _, t := range tasks {
						if t.Name == crewTask.Name {
							writeYAML("tasks", t.Name, t.TaskConfig)
							break
						}
					}
				}

				jsonResponse(w, http.StatusOK, map[string]string{"status": "imported", "name": c.Name})
				return
			}
		}
		jsonError(w, http.StatusNotFound, "template crew not found")

	default:
		jsonError(w, http.StatusBadRequest, "type must be agent, task, or crew")
	}
}

// containsQuery checks if any of the fields contain the query string.
func containsQuery(fields ...string) bool {
	if len(fields) == 0 {
		return false
	}
	query := fields[len(fields)-1]
	for _, f := range fields[:len(fields)-1] {
		if strings.Contains(strings.ToLower(f), query) {
			return true
		}
	}
	return false
}
