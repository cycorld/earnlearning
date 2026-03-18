package persistence

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type KanbanTask struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Filename    string `json:"filename"`
}

type TaskRepo struct {
	basePath string
}

func NewTaskRepo(basePath string) *TaskRepo {
	return &TaskRepo{basePath: basePath}
}

var taskStatuses = []string{"backlog", "todo", "in-progress", "done"}

func (r *TaskRepo) List() ([]*KanbanTask, error) {
	var tasks []*KanbanTask

	for _, status := range taskStatuses {
		dir := filepath.Join(r.basePath, status)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue // folder may not exist
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			task, err := r.parseTaskFile(filepath.Join(dir, entry.Name()), status)
			if err != nil {
				continue
			}
			tasks = append(tasks, task)
		}
	}

	if tasks == nil {
		tasks = []*KanbanTask{}
	}
	return tasks, nil
}

func (r *TaskRepo) parseTaskFile(path, status string) (*KanbanTask, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	task := &KanbanTask{
		Status:   status,
		Filename: filepath.Base(path),
		Priority: "medium",
		Type:     "feat",
	}

	scanner := bufio.NewScanner(f)
	inFrontmatter := false
	frontmatterDone := false
	var bodyLines []string

	for scanner.Scan() {
		line := scanner.Text()

		if !inFrontmatter && !frontmatterDone && strings.TrimSpace(line) == "---" {
			inFrontmatter = true
			continue
		}
		if inFrontmatter && strings.TrimSpace(line) == "---" {
			inFrontmatter = false
			frontmatterDone = true
			continue
		}

		if inFrontmatter {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			switch key {
			case "id":
				task.ID = val
			case "title":
				task.Title = val
			case "priority":
				task.Priority = val
			case "type":
				task.Type = val
			}
		} else if frontmatterDone {
			bodyLines = append(bodyLines, line)
		}
	}

	// Use first heading as title if frontmatter title is empty
	if task.Title == "" {
		for _, line := range bodyLines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "# ") {
				task.Title = strings.TrimPrefix(trimmed, "# ")
				break
			}
		}
	}
	if task.Title == "" {
		task.Title = strings.TrimSuffix(filepath.Base(path), ".md")
	}

	// ID from filename if not in frontmatter
	if task.ID == "" {
		name := filepath.Base(path)
		parts := strings.SplitN(name, "-", 2)
		if len(parts) > 0 {
			task.ID = parts[0]
		}
	}

	task.Description = strings.TrimSpace(strings.Join(bodyLines, "\n"))

	return task, nil
}
