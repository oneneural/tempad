// Package prompt handles rendering Liquid templates for issue prompts.
// See Spec Section 6.4 and Section 14.
package prompt

import (
	"fmt"
	"time"

	"github.com/osteele/liquid"

	"github.com/oneneural/tempad/internal/domain"
)

// DefaultPrompt is used when the workflow prompt body is empty.
const DefaultPrompt = "Work on issue {{ issue.identifier }}: {{ issue.title }}"

// TemplateError categorizes template failures per Spec Section 6.5.
type TemplateError struct {
	Kind    string // "template_parse_error" or "template_render_error"
	Message string
	Cause   error
}

func (e *TemplateError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Kind, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.Message)
}

func (e *TemplateError) Unwrap() error {
	return e.Cause
}

// Builder renders Liquid templates with issue data.
type Builder struct {
	engine *liquid.Engine
}

// NewBuilder creates a prompt builder with a configured Liquid engine.
func NewBuilder() *Builder {
	engine := liquid.NewEngine()
	return &Builder{engine: engine}
}

// Render renders a Liquid template string with issue data and optional
// attempt number.
//
// Template input variables:
//   - issue (object): all normalized issue fields
//   - attempt (int or nil): nil on first attempt, int on retry/continuation
//
// Returns the rendered string or a TemplateError.
func (b *Builder) Render(templateStr string, issue domain.Issue, attempt *int) (string, error) {
	if templateStr == "" {
		templateStr = DefaultPrompt
	}

	bindings := map[string]any{
		"issue": issueToMap(issue),
	}
	if attempt != nil {
		bindings["attempt"] = *attempt
	}

	out, err := b.engine.ParseAndRenderString(templateStr, bindings)
	if err != nil {
		return "", &TemplateError{
			Kind:    "template_render_error",
			Message: "failed to render prompt template",
			Cause:   err,
		}
	}

	return out, nil
}

// issueToMap converts a domain.Issue struct to a map[string]any so Liquid
// can access fields like issue.identifier, issue.labels, etc.
func issueToMap(issue domain.Issue) map[string]any {
	m := map[string]any{
		"id":          issue.ID,
		"identifier":  issue.Identifier,
		"title":       issue.Title,
		"description": issue.Description,
		"state":       issue.State,
		"assignee":    issue.Assignee,
		"branch_name": issue.BranchName,
		"url":         issue.URL,
		"labels":      issue.Labels,
	}

	// Priority: integer or nil.
	if issue.Priority != nil {
		m["priority"] = *issue.Priority
	} else {
		m["priority"] = nil
	}

	// Blocked_by: list of maps.
	blockers := make([]map[string]any, len(issue.BlockedBy))
	for i, b := range issue.BlockedBy {
		blockers[i] = map[string]any{
			"id":         b.ID,
			"identifier": b.Identifier,
			"state":      b.State,
		}
	}
	m["blocked_by"] = blockers

	// Timestamps as ISO-8601 strings.
	if issue.CreatedAt != nil {
		m["created_at"] = issue.CreatedAt.Format(time.RFC3339)
	} else {
		m["created_at"] = nil
	}
	if issue.UpdatedAt != nil {
		m["updated_at"] = issue.UpdatedAt.Format(time.RFC3339)
	} else {
		m["updated_at"] = nil
	}

	return m
}
