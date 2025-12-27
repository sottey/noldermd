package api

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	taskLinePrefix = regexp.MustCompile(`^\s*\*\s+`)
	taskProject    = regexp.MustCompile(`(^|\s)\+([A-Za-z0-9][A-Za-z0-9_-]*)\b`)
	taskDueDate    = regexp.MustCompile(`(^|\s)>(\d{4}-\d{2}-\d{2})\b`)
	taskPriority   = regexp.MustCompile(`(^|\s)-([1-5])\b`)
	taskTag        = regexp.MustCompile(`(^|\s)#([A-Za-z]+)\b`)
)

type ParsedNoteTask struct {
	LineNumber int
	LineHash   string
	Title      string
	Project    string
	DueDate    string
	Priority   int
	Tags       []string
}

func parseNoteTasks(content string) []ParsedNoteTask {
	lines := strings.Split(content, "\n")
	tasks := make([]ParsedNoteTask, 0)
	for i, line := range lines {
		raw := strings.TrimSuffix(line, "\r")
		loc := taskLinePrefix.FindStringIndex(raw)
		if loc == nil {
			continue
		}
		rest := raw[loc[1]:]
		if strings.TrimSpace(rest) == "" {
			continue
		}

		project := ""
		if match := taskProject.FindStringSubmatch(rest); len(match) > 0 {
			project = match[2]
		}

		dueDate := ""
		if match := taskDueDate.FindStringSubmatch(rest); len(match) > 0 {
			if _, err := time.Parse(dueDateLayout, match[2]); err == nil {
				dueDate = match[2]
			}
		}

		priority := 3
		if match := taskPriority.FindStringSubmatch(rest); len(match) > 0 {
			if value, err := strconv.Atoi(match[2]); err == nil {
				priority = value
			}
		}

		tags := extractTaskTags(rest)
		tasks = append(tasks, ParsedNoteTask{
			LineNumber: i + 1,
			LineHash:   hashLine(raw),
			Title:      rest,
			Project:    project,
			DueDate:    dueDate,
			Priority:   priority,
			Tags:       tags,
		})
	}
	return tasks
}

func extractTaskTags(text string) []string {
	if text == "" {
		return []string{}
	}
	matches := taskTag.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{})
	tags := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		tag := strings.ToLower(match[2])
		if tag == "" {
			continue
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
	}
	return tags
}

func hashLine(line string) string {
	sum := sha256.Sum256([]byte(line))
	return hex.EncodeToString(sum[:])
}
