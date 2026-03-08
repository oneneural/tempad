package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// WorkflowDefinition is the parsed WORKFLOW.md payload.
// See Spec Section 4.1.2.
type WorkflowDefinition struct {
	// Config is the YAML front matter root object (map of settings).
	Config map[string]any

	// PromptTemplate is the trimmed Markdown body after front matter.
	PromptTemplate string
}

// WorkflowError categorizes workflow loading failures per Spec Section 6.5.
type WorkflowError struct {
	Kind    string // "missing_workflow_file", "workflow_parse_error", "workflow_front_matter_not_a_map"
	Message string
	Cause   error
}

func (e *WorkflowError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Kind, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.Message)
}

func (e *WorkflowError) Unwrap() error {
	return e.Cause
}

// LoadWorkflow loads and parses a WORKFLOW.md file.
// See Spec Section 6.1, 6.2, 6.3, 6.5.
//
// Path precedence:
//  1. Explicit path argument (if non-empty).
//  2. Default: "./WORKFLOW.md" in the current working directory.
//
// Parsing rules:
//   - If the file starts with "---", parse lines until the next "---" as
//     YAML front matter. Remaining lines become the prompt body.
//   - If front matter is absent, the entire file is treated as prompt body
//     with an empty config map.
//   - YAML front matter must decode to a map; non-map YAML is an error.
//   - Prompt body is trimmed before use.
//   - Unknown keys in front matter are ignored (forward compatibility).
func LoadWorkflow(path string) (*WorkflowDefinition, error) {
	if path == "" {
		path = "WORKFLOW.md"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &WorkflowError{
				Kind:    "missing_workflow_file",
				Message: fmt.Sprintf("workflow file not found: %s", path),
				Cause:   err,
			}
		}
		return nil, &WorkflowError{
			Kind:    "missing_workflow_file",
			Message: fmt.Sprintf("cannot read workflow file: %s", path),
			Cause:   err,
		}
	}

	return parseWorkflow(string(data))
}

// parseWorkflow splits YAML front matter from the markdown body and parses
// the front matter into a config map.
func parseWorkflow(content string) (*WorkflowDefinition, error) {
	frontMatter, body, hasFrontMatter := splitFrontMatter(content)

	wf := &WorkflowDefinition{
		Config:         make(map[string]any),
		PromptTemplate: strings.TrimSpace(body),
	}

	if !hasFrontMatter {
		return wf, nil
	}

	// Parse the YAML front matter.
	var raw any
	if err := yaml.Unmarshal([]byte(frontMatter), &raw); err != nil {
		return nil, &WorkflowError{
			Kind:    "workflow_parse_error",
			Message: "failed to parse YAML front matter",
			Cause:   err,
		}
	}

	// Front matter must be a map.
	if raw == nil {
		// Empty front matter (just "---\n---") → empty config map.
		return wf, nil
	}

	configMap, ok := raw.(map[string]any)
	if !ok {
		return nil, &WorkflowError{
			Kind:    "workflow_front_matter_not_a_map",
			Message: fmt.Sprintf("front matter must be a YAML map, got %T", raw),
		}
	}

	wf.Config = configMap
	return wf, nil
}

// splitFrontMatter splits content at the YAML front matter delimiters (---).
// Returns the front matter content, the body, and whether front matter was found.
func splitFrontMatter(content string) (frontMatter, body string, found bool) {
	content = strings.TrimSpace(content)

	if !strings.HasPrefix(content, "---") {
		return "", content, false
	}

	// Find the closing "---" after the opening one.
	// The opening "---" is at the start. Find the next one.
	rest := content[3:] // skip opening "---"

	// Skip the rest of the first line (may have trailing whitespace/newline).
	if idx := strings.IndexByte(rest, '\n'); idx >= 0 {
		rest = rest[idx+1:]
	} else {
		// Only "---" with no newline → no closing delimiter.
		return "", content, false
	}

	// Find closing "---".
	closingIdx := strings.Index(rest, "---")
	if closingIdx < 0 {
		// No closing delimiter → treat entire file as body.
		return "", content, false
	}

	// Verify the closing "---" is at the start of a line.
	if closingIdx > 0 && rest[closingIdx-1] != '\n' {
		return "", content, false
	}

	frontMatter = rest[:closingIdx]
	body = rest[closingIdx+3:]

	return strings.TrimSpace(frontMatter), body, true
}

// getNestedString extracts a nested string value from a config map.
// path is a dot-separated key path like "tracker.kind".
func getNestedString(m map[string]any, path string) (string, bool) {
	parts := strings.Split(path, ".")
	current := any(m)

	for i, part := range parts {
		cm, ok := current.(map[string]any)
		if !ok {
			return "", false
		}
		val, exists := cm[part]
		if !exists {
			return "", false
		}
		if i == len(parts)-1 {
			switch v := val.(type) {
			case string:
				return v, true
			case int:
				return fmt.Sprintf("%d", v), true
			case float64:
				// YAML integers sometimes parse as float64.
				if v == float64(int(v)) {
					return fmt.Sprintf("%d", int(v)), true
				}
				return fmt.Sprintf("%g", v), true
			default:
				return "", false
			}
		}
		current = val
	}
	return "", false
}

// getNestedInt extracts a nested integer value from a config map.
func getNestedInt(m map[string]any, path string) (int, bool) {
	parts := strings.Split(path, ".")
	current := any(m)

	for i, part := range parts {
		cm, ok := current.(map[string]any)
		if !ok {
			return 0, false
		}
		val, exists := cm[part]
		if !exists {
			return 0, false
		}
		if i == len(parts)-1 {
			switch v := val.(type) {
			case int:
				return v, true
			case float64:
				return int(v), true
			case string:
				// Try parsing string integers (Spec: "integer or string integer").
				var n int
				if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
					return n, true
				}
				return 0, false
			default:
				return 0, false
			}
		}
		current = val
	}
	return 0, false
}

// getNestedStringList extracts a nested list of strings or a comma-separated
// string from a config map. See Spec Section 6.3.1 (active_states, terminal_states).
func getNestedStringList(m map[string]any, path string) ([]string, bool) {
	parts := strings.Split(path, ".")
	current := any(m)

	for i, part := range parts {
		cm, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		val, exists := cm[part]
		if !exists {
			return nil, false
		}
		if i == len(parts)-1 {
			switch v := val.(type) {
			case []any:
				result := make([]string, 0, len(v))
				for _, item := range v {
					if s, ok := item.(string); ok {
						result = append(result, s)
					}
				}
				return result, true
			case string:
				// Comma-separated string.
				parts := strings.Split(v, ",")
				result := make([]string, 0, len(parts))
				for _, p := range parts {
					trimmed := strings.TrimSpace(p)
					if trimmed != "" {
						result = append(result, trimmed)
					}
				}
				return result, true
			default:
				return nil, false
			}
		}
		current = val
	}
	return nil, false
}

// getNestedMap extracts a nested map[string]int from a config map.
// Used for agent.max_concurrent_by_state.
func getNestedIntMap(m map[string]any, path string) (map[string]int, bool) {
	parts := strings.Split(path, ".")
	current := any(m)

	for i, part := range parts {
		cm, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		val, exists := cm[part]
		if !exists {
			return nil, false
		}
		if i == len(parts)-1 {
			cm2, ok := val.(map[string]any)
			if !ok {
				return nil, false
			}
			result := make(map[string]int, len(cm2))
			for k, v := range cm2 {
				switch n := v.(type) {
				case int:
					if n > 0 {
						result[k] = n
					}
				case float64:
					if int(n) > 0 {
						result[k] = int(n)
					}
				}
			}
			return result, true
		}
		current = val
	}
	return nil, false
}
