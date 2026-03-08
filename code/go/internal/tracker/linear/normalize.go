package linear

import (
	"strings"
	"time"

	"github.com/oneneural/tempad/internal/domain"
)

// normalizeIssue converts a Linear API issueNode into a domain.Issue.
func normalizeIssue(raw issueNode) domain.Issue {
	issue := domain.Issue{
		ID:         raw.ID,
		Identifier: raw.Identifier,
		Title:      raw.Title,
		Description: raw.Description,
		Priority:   raw.Priority,
		State:      raw.State.Name,
		URL:        raw.URL,
	}

	// BranchName may be nil.
	if raw.BranchName != nil {
		issue.BranchName = *raw.BranchName
	}

	// Assignee: use email if available, fall back to ID.
	if raw.Assignee != nil {
		if raw.Assignee.Email != "" {
			issue.Assignee = raw.Assignee.Email
		} else {
			issue.Assignee = raw.Assignee.ID
		}
	}

	// Labels: lowercase all names.
	if len(raw.Labels.Nodes) > 0 {
		issue.Labels = make([]string, len(raw.Labels.Nodes))
		for i, l := range raw.Labels.Nodes {
			issue.Labels[i] = strings.ToLower(l.Name)
		}
	}

	// BlockedBy: derive from relations where type is "is-blocked-by".
	for _, rel := range raw.Relations.Nodes {
		if rel.Type == "blocks" {
			issue.BlockedBy = append(issue.BlockedBy, domain.BlockerRef{
				ID:         rel.RelatedIssue.ID,
				Identifier: rel.RelatedIssue.Identifier,
				State:      rel.RelatedIssue.State.Name,
			})
		}
	}

	// Timestamps: parse ISO-8601.
	if t, err := time.Parse(time.RFC3339, raw.CreatedAt); err == nil {
		issue.CreatedAt = &t
	}
	if t, err := time.Parse(time.RFC3339, raw.UpdatedAt); err == nil {
		issue.UpdatedAt = &t
	}

	return issue
}

// normalizeIssues converts a slice of issueNodes to domain.Issues.
func normalizeIssues(nodes []issueNode) []domain.Issue {
	issues := make([]domain.Issue, len(nodes))
	for i, n := range nodes {
		issues[i] = normalizeIssue(n)
	}
	return issues
}
