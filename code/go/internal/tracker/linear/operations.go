package linear

import (
	"context"
	"fmt"

	"github.com/oneneural/tempad/internal/domain"
	"github.com/oneneural/tempad/internal/tracker"
)

// Compile-time check that LinearClient implements tracker.Client.
var _ tracker.Client = (*LinearClient)(nil)

// ResolveIdentity resolves the configured email to a Linear user ID.
// This should be called once after construction; the result is cached.
func (c *LinearClient) ResolveIdentity(ctx context.Context) error {
	if c.identity == "" {
		return &tracker.MissingTrackerIdentityError{}
	}

	var data usersData
	err := c.do(ctx, userByEmailQuery, map[string]any{
		"email": c.identity,
	}, &data)
	if err != nil {
		return fmt.Errorf("resolve identity %q: %w", c.identity, err)
	}

	if len(data.Users.Nodes) == 0 {
		return fmt.Errorf("no Linear user found for email %q", c.identity)
	}

	c.userID = data.Users.Nodes[0].ID
	return nil
}

// FetchCandidateIssues returns unassigned issues in active states plus
// issues assigned to the current user (for resumption), deduplicated.
func (c *LinearClient) FetchCandidateIssues(ctx context.Context) ([]domain.Issue, error) {
	extractIssues := func(data *issuesData) page[issueNode] {
		return page[issueNode]{
			Nodes:    data.Issues.Nodes,
			PageInfo: data.Issues.PageInfo,
		}
	}

	// Fetch unassigned candidates.
	unassigned, err := fetchAll[issuesData, issueNode](ctx, c, candidateIssuesQuery, map[string]any{
		"projectSlug": c.projectSlug,
		"states":      c.activeStates,
	}, extractIssues)
	if err != nil {
		return nil, fmt.Errorf("fetch unassigned candidates: %w", err)
	}

	// Fetch my assigned issues (resumption).
	var myAssigned []issueNode
	if c.userID != "" {
		myAssigned, err = fetchAll[issuesData, issueNode](ctx, c, assignedToMeQuery, map[string]any{
			"projectSlug": c.projectSlug,
			"states":      c.activeStates,
			"userID":      c.userID,
		}, extractIssues)
		if err != nil {
			return nil, fmt.Errorf("fetch assigned issues: %w", err)
		}
	}

	// Merge and deduplicate by ID.
	all := make([]issueNode, 0, len(unassigned)+len(myAssigned))
	all = append(all, unassigned...)
	all = append(all, myAssigned...)

	seen := make(map[string]bool, len(all))
	deduped := make([]issueNode, 0, len(all))
	for _, n := range all {
		if !seen[n.ID] {
			seen[n.ID] = true
			deduped = append(deduped, n)
		}
	}

	return normalizeIssues(deduped), nil
}

// FetchIssuesByStates returns issues in the given states (e.g. for terminal cleanup).
func (c *LinearClient) FetchIssuesByStates(ctx context.Context, states []string) ([]domain.Issue, error) {
	nodes, err := fetchAll[issuesData, issueNode](ctx, c, issuesByStatesQuery, map[string]any{
		"projectSlug": c.projectSlug,
		"states":      states,
	}, func(data *issuesData) page[issueNode] {
		return page[issueNode]{
			Nodes:    data.Issues.Nodes,
			PageInfo: data.Issues.PageInfo,
		}
	})
	if err != nil {
		return nil, fmt.Errorf("fetch issues by states: %w", err)
	}

	return normalizeIssues(nodes), nil
}

// FetchIssueStatesByIDs returns a map of issue ID → current state name.
func (c *LinearClient) FetchIssueStatesByIDs(ctx context.Context, ids []string) (map[string]string, error) {
	var data nodesData
	err := c.do(ctx, issueStatesByIDsQuery, map[string]any{
		"ids": ids,
	}, &data)
	if err != nil {
		return nil, fmt.Errorf("fetch issue states: %w", err)
	}

	result := make(map[string]string, len(data.Nodes))
	for _, node := range data.Nodes {
		if node.ID != "" && node.State != nil {
			result[node.ID] = node.State.Name
		}
	}

	return result, nil
}

// FetchIssue returns a single issue by ID.
func (c *LinearClient) FetchIssue(ctx context.Context, id string) (*domain.Issue, error) {
	var data singleIssueData
	err := c.do(ctx, singleIssueQuery, map[string]any{
		"id": id,
	}, &data)
	if err != nil {
		return nil, fmt.Errorf("fetch issue %s: %w", id, err)
	}

	issue := normalizeIssue(data.Issue)
	return &issue, nil
}

// AssignIssue assigns the issue to the given identity. The identity parameter
// is used to resolve the user ID if not already cached.
func (c *LinearClient) AssignIssue(ctx context.Context, issueID string, identity string) error {
	assigneeID := c.userID
	if assigneeID == "" {
		return &tracker.MissingTrackerIdentityError{}
	}

	var data assignIssueData
	err := c.do(ctx, assignIssueMutation, map[string]any{
		"issueID":    issueID,
		"assigneeID": assigneeID,
	}, &data)
	if err != nil {
		return fmt.Errorf("assign issue %s: %w", issueID, err)
	}

	if !data.IssueUpdate.Success {
		return fmt.Errorf("assign issue %s: mutation returned success=false", issueID)
	}

	return nil
}

// UnassignIssue removes assignment from the issue.
func (c *LinearClient) UnassignIssue(ctx context.Context, issueID string) error {
	var data unassignIssueData
	err := c.do(ctx, unassignIssueMutation, map[string]any{
		"issueID": issueID,
	}, &data)
	if err != nil {
		return fmt.Errorf("unassign issue %s: %w", issueID, err)
	}

	if !data.IssueUpdate.Success {
		return fmt.Errorf("unassign issue %s: mutation returned success=false", issueID)
	}

	return nil
}
