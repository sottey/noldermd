package api

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"time"
)

var (
	todoLinePattern      = regexp.MustCompile(`^\s*-\s+\[( |x|X|✓)\]\s+`)
	todoTogglePattern    = regexp.MustCompile(`^(\s*-\s+\[)( |x|X|✓)(\]\s+)`)
	todoCompletedPattern = regexp.MustCompile(`^\s*-\s+\[(x|X|✓)\]\s+`)
	taskProjectPattern   = regexp.MustCompile(`(^|\s)\+([A-Za-z]+)\b`)
	taskTagPattern       = regexp.MustCompile(`(^|\s)#([A-Za-z]+)\b`)
	taskMentionPattern   = regexp.MustCompile(`(^|\s)@([A-Za-z]+)\b`)
	taskDuePattern       = regexp.MustCompile(`(^|\s)>(\S+)`)
	taskPriorityPattern  = regexp.MustCompile(`(^|\s)\^([1-5])\b`)
	taskTokenPattern     = regexp.MustCompile(`(^|\s)(#[A-Za-z]+|@[A-Za-z]+|\+[A-Za-z]+|\^[1-5]|>\S+)`)
)

type ParsedTodo struct {
	LineNumber   int
	LineHash     string
	Text         string
	Completed    bool
	Project      string
	Tags         []string
	Mentions     []string
	DueDateRaw   string
	DueDateISO   string
	DueDateValid bool
	Priority     int
}

func parseTodoLines(content string) []ParsedTodo {
	lines := strings.Split(content, "\n")
	todos := make([]ParsedTodo, 0)
	for i, line := range lines {
		raw := strings.TrimSuffix(line, "\r")
		loc := todoLinePattern.FindStringIndex(raw)
		if loc == nil {
			continue
		}
		match := todoLinePattern.FindStringSubmatch(raw)
		if len(match) < 2 {
			continue
		}
		rest := raw[loc[1]:]
		if strings.TrimSpace(rest) == "" {
			continue
		}

		completed := match[1] != " "
		project := extractFirstMatch(taskProjectPattern, rest)
		tags := extractMatches(taskTagPattern, rest)
		mentions := extractMatches(taskMentionPattern, rest)
		priority := extractPriority(rest)
		dueRaw := extractDueDate(rest)
		dueISO, dueValid := normalizeDueDate(dueRaw)

		text := cleanTaskText(rest)
		if text == "" {
			text = strings.TrimSpace(rest)
		}

		todos = append(todos, ParsedTodo{
			LineNumber:   i + 1,
			LineHash:     hashLine(raw),
			Text:         text,
			Completed:    completed,
			Project:      strings.ToLower(project),
			Tags:         tags,
			Mentions:     mentions,
			DueDateRaw:   dueRaw,
			DueDateISO:   dueISO,
			DueDateValid: dueValid,
			Priority:     priority,
		})
	}
	return todos
}

func setTaskLineCompletion(line string, completed bool) (string, bool) {
	match := todoTogglePattern.FindStringSubmatchIndex(line)
	if match == nil || len(match) < 6 {
		return "", false
	}
	marker := " "
	if completed {
		marker = "x"
	}
	updated := line[:match[4]] + marker + line[match[5]:]
	return updated, true
}

func archiveCompletedTaskLine(line string) (string, bool) {
	if !todoCompletedPattern.MatchString(line) {
		return "", false
	}
	trimmed := strings.TrimSuffix(line, "\r")
	ending := ""
	if trimmed != line {
		ending = "\r"
	}
	indent := leadingWhitespace(trimmed)
	rest := strings.TrimPrefix(trimmed, indent)
	updated := indent + "~ " + rest + ending
	return updated, true
}

func leadingWhitespace(text string) string {
	for i, r := range text {
		if r != ' ' && r != '\t' {
			return text[:i]
		}
	}
	return text
}

func extractFirstMatch(pattern *regexp.Regexp, text string) string {
	match := pattern.FindStringSubmatch(text)
	if len(match) < 3 {
		return ""
	}
	return match[2]
}

func extractMatches(pattern *regexp.Regexp, text string) []string {
	matches := pattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{})
	values := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		value := strings.ToLower(match[2])
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values
}

func extractPriority(text string) int {
	match := taskPriorityPattern.FindStringSubmatch(text)
	if len(match) < 3 {
		return 0
	}
	switch match[2] {
	case "1":
		return 1
	case "2":
		return 2
	case "3":
		return 3
	case "4":
		return 4
	case "5":
		return 5
	default:
		return 0
	}
}

func extractDueDate(text string) string {
	match := taskDuePattern.FindStringSubmatch(text)
	if len(match) < 3 {
		return ""
	}
	return strings.TrimRight(match[2], ".,;:)]}")
}

func normalizeDueDate(raw string) (string, bool) {
	if raw == "" {
		return "", false
	}
	layouts := []string{
		"2006-01-02",
		"2006/01/02",
		"2006.01.02",
		"01/02/2006",
		"02/01/2006",
		"Jan 2 2006",
		"January 2 2006",
		"Jan 2, 2006",
		"January 2, 2006",
		time.RFC3339,
		time.RFC3339Nano,
	}
	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, raw, time.Local)
		if err == nil {
			return parsed.Format("2006-01-02"), true
		}
	}
	return "", false
}

func cleanTaskText(text string) string {
	cleaned := taskTokenPattern.ReplaceAllString(text, " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	return strings.TrimSpace(cleaned)
}

func hashLine(line string) string {
	sum := sha256.Sum256([]byte(line))
	return hex.EncodeToString(sum[:])
}
