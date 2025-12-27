package api

import "time"

func (s *Server) syncTasksFromNote(notePath, content string) error {
	parsed := parseNoteTasks(content)
	if len(parsed) == 0 {
		return nil
	}

	store, _, err := s.loadTasks()
	if err != nil {
		return err
	}

	lineIndex := make(map[int]int)
	hashIndex := make(map[string]int)
	for i, task := range store.Tasks {
		if task.Source == nil || task.Source.NotePath != notePath {
			continue
		}
		if task.Source.LineNumber > 0 {
			lineIndex[task.Source.LineNumber] = i
		}
		if task.Source.LineHash != "" {
			hashIndex[task.Source.LineHash] = i
		}
	}

	now := time.Now().UTC()
	used := make(map[int]bool)

	for _, parsedTask := range parsed {
		idx, ok := lineIndex[parsedTask.LineNumber]
		if ok && used[idx] {
			ok = false
		}
		if !ok {
			if fallback, okHash := hashIndex[parsedTask.LineHash]; okHash && !used[fallback] {
				idx = fallback
				ok = true
			}
		}

		if ok {
			updated := store.Tasks[idx]
			updated.Title = parsedTask.Title
			updated.Project = parsedTask.Project
			updated.Tags = parsedTask.Tags
			updated.DueDate = parsedTask.DueDate
			updated.Priority = parsedTask.Priority
			updated.Updated = now
			updated.Source = &TaskSource{
				NotePath:   notePath,
				LineNumber: parsedTask.LineNumber,
				LineHash:   parsedTask.LineHash,
			}
			store.Tasks[idx] = updated
			used[idx] = true
			continue
		}

		store.Tasks = append(store.Tasks, Task{
			ID:        newUUID(),
			Title:     parsedTask.Title,
			Project:   parsedTask.Project,
			Tags:      parsedTask.Tags,
			Created:   now,
			Updated:   now,
			DueDate:   parsedTask.DueDate,
			Priority:  parsedTask.Priority,
			Completed: false,
			Notes:     "",
			Recurring: nil,
			Source: &TaskSource{
				NotePath:   notePath,
				LineNumber: parsedTask.LineNumber,
				LineHash:   parsedTask.LineHash,
			},
		})
	}

	return s.saveTasks(store)
}
