package linear

import (
	"testing"
)

func ptrString(s string) *string { return &s }
func ptrInt(i int) *int          { return &i }

func TestNormalizeIssue_AllFields(t *testing.T) {
	raw := issueNode{
		ID:          "uuid-1",
		Identifier:  "ABC-123",
		Title:       "Fix login",
		Description: "Users can't log in",
		Priority:    ptrInt(2),
		BranchName:  ptrString("abc-123-fix-login"),
		URL:         "https://linear.app/team/issue/ABC-123",
		State:       stateNode{Name: "In Progress"},
		Assignee:    &assigneeNode{ID: "user-1", Email: "dev@example.com"},
		Labels: labelsConn{Nodes: []labelNode{
			{Name: "Bug"},
			{Name: "Frontend"},
		}},
		Relations: relationsConn{Nodes: []relationNode{
			{
				Type: "blocks",
				RelatedIssue: relatedIssueNode{
					ID: "blocker-1", Identifier: "XYZ-001",
					State: stateNode{Name: "Todo"},
				},
			},
			{
				Type:         "related",
				RelatedIssue: relatedIssueNode{ID: "rel-1", Identifier: "REL-001"},
			},
		}},
		CreatedAt: "2026-03-08T12:00:00Z",
		UpdatedAt: "2026-03-08T14:00:00Z",
	}

	issue := normalizeIssue(raw)

	if issue.ID != "uuid-1" {
		t.Errorf("ID: got %q", issue.ID)
	}
	if issue.Identifier != "ABC-123" {
		t.Errorf("Identifier: got %q", issue.Identifier)
	}
	if issue.State != "In Progress" {
		t.Errorf("State: got %q", issue.State)
	}
	if issue.Priority == nil || *issue.Priority != 2 {
		t.Errorf("Priority: got %v", issue.Priority)
	}
	if issue.BranchName != "abc-123-fix-login" {
		t.Errorf("BranchName: got %q", issue.BranchName)
	}
	if issue.Assignee != "dev@example.com" {
		t.Errorf("Assignee: got %q", issue.Assignee)
	}
	if issue.URL != "https://linear.app/team/issue/ABC-123" {
		t.Errorf("URL: got %q", issue.URL)
	}
}

func TestNormalizeIssue_LabelsLowercase(t *testing.T) {
	raw := issueNode{
		Labels: labelsConn{Nodes: []labelNode{
			{Name: "Bug"},
			{Name: "Frontend"},
			{Name: "P1"},
		}},
	}

	issue := normalizeIssue(raw)

	expected := []string{"bug", "frontend", "p1"}
	if len(issue.Labels) != len(expected) {
		t.Fatalf("expected %d labels, got %d", len(expected), len(issue.Labels))
	}
	for i, want := range expected {
		if issue.Labels[i] != want {
			t.Errorf("label[%d]: got %q, want %q", i, issue.Labels[i], want)
		}
	}
}

func TestNormalizeIssue_BlockedByFromRelations(t *testing.T) {
	raw := issueNode{
		Relations: relationsConn{Nodes: []relationNode{
			{
				Type: "blocks",
				RelatedIssue: relatedIssueNode{
					ID: "b1", Identifier: "BLK-1", State: stateNode{Name: "Todo"},
				},
			},
			{
				Type:         "related",
				RelatedIssue: relatedIssueNode{ID: "r1", Identifier: "REL-1"},
			},
			{
				Type: "blocks",
				RelatedIssue: relatedIssueNode{
					ID: "b2", Identifier: "BLK-2", State: stateNode{Name: "Done"},
				},
			},
		}},
	}

	issue := normalizeIssue(raw)

	if len(issue.BlockedBy) != 2 {
		t.Fatalf("expected 2 blockers, got %d", len(issue.BlockedBy))
	}
	if issue.BlockedBy[0].Identifier != "BLK-1" {
		t.Errorf("blocker[0]: got %q", issue.BlockedBy[0].Identifier)
	}
	if issue.BlockedBy[1].Identifier != "BLK-2" {
		t.Errorf("blocker[1]: got %q", issue.BlockedBy[1].Identifier)
	}
}

func TestNormalizeIssue_NilPriority(t *testing.T) {
	raw := issueNode{Priority: nil}
	issue := normalizeIssue(raw)
	if issue.Priority != nil {
		t.Errorf("expected nil priority, got %v", issue.Priority)
	}
}

func TestNormalizeIssue_NilBranchName(t *testing.T) {
	raw := issueNode{BranchName: nil}
	issue := normalizeIssue(raw)
	if issue.BranchName != "" {
		t.Errorf("expected empty branch name, got %q", issue.BranchName)
	}
}

func TestNormalizeIssue_NilAssignee(t *testing.T) {
	raw := issueNode{Assignee: nil}
	issue := normalizeIssue(raw)
	if issue.Assignee != "" {
		t.Errorf("expected empty assignee, got %q", issue.Assignee)
	}
}

func TestNormalizeIssue_AssigneeFallbackToID(t *testing.T) {
	raw := issueNode{Assignee: &assigneeNode{ID: "user-id-only", Email: ""}}
	issue := normalizeIssue(raw)
	if issue.Assignee != "user-id-only" {
		t.Errorf("expected assignee=user-id-only, got %q", issue.Assignee)
	}
}

func TestNormalizeIssue_EmptyLabels(t *testing.T) {
	raw := issueNode{}
	issue := normalizeIssue(raw)
	if issue.Labels != nil {
		t.Errorf("expected nil labels, got %v", issue.Labels)
	}
}

func TestNormalizeIssue_EmptyRelations(t *testing.T) {
	raw := issueNode{}
	issue := normalizeIssue(raw)
	if len(issue.BlockedBy) != 0 {
		t.Errorf("expected no blockers, got %d", len(issue.BlockedBy))
	}
}

func TestNormalizeIssue_Timestamps(t *testing.T) {
	raw := issueNode{
		CreatedAt: "2026-03-08T12:00:00Z",
		UpdatedAt: "2026-03-08T14:30:00Z",
	}
	issue := normalizeIssue(raw)

	if issue.CreatedAt == nil {
		t.Fatal("expected CreatedAt to be set")
	}
	if issue.CreatedAt.Hour() != 12 {
		t.Errorf("CreatedAt hour: got %d", issue.CreatedAt.Hour())
	}
	if issue.UpdatedAt == nil {
		t.Fatal("expected UpdatedAt to be set")
	}
	if issue.UpdatedAt.Minute() != 30 {
		t.Errorf("UpdatedAt minute: got %d", issue.UpdatedAt.Minute())
	}
}

func TestNormalizeIssue_InvalidTimestamps(t *testing.T) {
	raw := issueNode{
		CreatedAt: "not-a-date",
		UpdatedAt: "",
	}
	issue := normalizeIssue(raw)

	if issue.CreatedAt != nil {
		t.Error("expected nil CreatedAt for invalid timestamp")
	}
	if issue.UpdatedAt != nil {
		t.Error("expected nil UpdatedAt for empty timestamp")
	}
}

func TestNormalizeIssues_Batch(t *testing.T) {
	nodes := []issueNode{
		{ID: "1", Identifier: "A-1", Title: "First"},
		{ID: "2", Identifier: "A-2", Title: "Second"},
		{ID: "3", Identifier: "A-3", Title: "Third"},
	}

	issues := normalizeIssues(nodes)

	if len(issues) != 3 {
		t.Fatalf("expected 3 issues, got %d", len(issues))
	}
	if issues[2].Title != "Third" {
		t.Errorf("expected Third, got %s", issues[2].Title)
	}
}
