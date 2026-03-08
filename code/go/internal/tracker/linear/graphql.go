// Package linear implements the tracker.Client interface for Linear's GraphQL API.
package linear

// GraphQL query and mutation string constants for Linear API operations.
// CRITICAL: All project filters use `project.slug` (NOT `slugId` — deprecated).

// candidateIssuesQuery fetches unassigned issues in active states for a project.
// Uses cursor-based pagination with first/after.
const candidateIssuesQuery = `
query CandidateIssues($projectSlug: String!, $states: [String!]!, $first: Int!, $after: String) {
  issues(
    filter: {
      project: { slug: { eq: $projectSlug } }
      state: { name: { in: $states } }
      assignee: { null: true }
    }
    first: $first
    after: $after
    orderBy: createdAt
  ) {
    nodes {
      ...IssueFields
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
` + issueFieldsFragment

// assignedToMeQuery fetches issues assigned to a specific user (for resumption).
const assignedToMeQuery = `
query AssignedToMe($projectSlug: String!, $states: [String!]!, $userID: String!, $first: Int!, $after: String) {
  issues(
    filter: {
      project: { slug: { eq: $projectSlug } }
      state: { name: { in: $states } }
      assignee: { id: { eq: $userID } }
    }
    first: $first
    after: $after
    orderBy: createdAt
  ) {
    nodes {
      ...IssueFields
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
` + issueFieldsFragment

// issuesByStatesQuery fetches issues in given states (for terminal cleanup).
const issuesByStatesQuery = `
query IssuesByStates($projectSlug: String!, $states: [String!]!, $first: Int!, $after: String) {
  issues(
    filter: {
      project: { slug: { eq: $projectSlug } }
      state: { name: { in: $states } }
    }
    first: $first
    after: $after
    orderBy: createdAt
  ) {
    nodes {
      ...IssueFields
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
` + issueFieldsFragment

// singleIssueQuery fetches a single issue by ID (for claim verification).
const singleIssueQuery = `
query SingleIssue($id: String!) {
  issue(id: $id) {
    ...IssueFields
  }
}
` + issueFieldsFragment

// issueStatesByIDsQuery fetches current states for a batch of issue IDs.
// Uses the nodes query for batch lookup.
const issueStatesByIDsQuery = `
query IssueStatesByIDs($ids: [ID!]!) {
  nodes(ids: $ids) {
    ... on Issue {
      id
      state {
        name
      }
    }
  }
}
`

// userByEmailQuery resolves a user email to a Linear user ID.
const userByEmailQuery = `
query UserByEmail($email: String!) {
  users(filter: { email: { eq: $email } }) {
    nodes {
      id
      email
    }
  }
}
`

// assignIssueMutation assigns an issue to a user.
const assignIssueMutation = `
mutation AssignIssue($issueID: String!, $assigneeID: String!) {
  issueUpdate(id: $issueID, input: { assigneeId: $assigneeID }) {
    success
    issue {
      id
      assignee {
        id
        email
      }
    }
  }
}
`

// unassignIssueMutation removes assignment from an issue.
const unassignIssueMutation = `
mutation UnassignIssue($issueID: String!) {
  issueUpdate(id: $issueID, input: { assigneeId: null }) {
    success
    issue {
      id
    }
  }
}
`

// issueFieldsFragment contains all fields needed for domain.Issue normalization.
const issueFieldsFragment = `
fragment IssueFields on Issue {
  id
  identifier
  title
  description
  priority
  branchName
  url
  state {
    name
  }
  assignee {
    id
    email
  }
  labels {
    nodes {
      name
    }
  }
  relations {
    nodes {
      type
      relatedIssue {
        id
        identifier
        state {
          name
        }
      }
    }
  }
  createdAt
  updatedAt
}
`

// --- Request/Response Types ---

// graphqlRequest is the JSON body sent to Linear's GraphQL endpoint.
type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// graphqlResponse is the top-level response envelope from Linear.
type graphqlResponse[T any] struct {
	Data   T              `json:"data"`
	Errors []graphqlError `json:"errors,omitempty"`
}

// graphqlError represents a single GraphQL error from Linear.
type graphqlError struct {
	Message    string         `json:"message"`
	Extensions map[string]any `json:"extensions,omitempty"`
}

// pageInfo contains cursor-based pagination metadata.
type pageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

// --- Issue Response Types ---

// issuesData wraps the issues query response.
type issuesData struct {
	Issues issuesConnection `json:"issues"`
}

// issuesConnection contains paginated issue nodes.
type issuesConnection struct {
	Nodes    []issueNode `json:"nodes"`
	PageInfo pageInfo    `json:"pageInfo"`
}

// issueNode represents a single issue from Linear's GraphQL response.
type issueNode struct {
	ID          string       `json:"id"`
	Identifier  string       `json:"identifier"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Priority    *int         `json:"priority"`
	BranchName  *string      `json:"branchName"`
	URL         string       `json:"url"`
	State       stateNode    `json:"state"`
	Assignee    *assigneeNode `json:"assignee"`
	Labels      labelsConn   `json:"labels"`
	Relations   relationsConn `json:"relations"`
	CreatedAt   string       `json:"createdAt"`
	UpdatedAt   string       `json:"updatedAt"`
}

type stateNode struct {
	Name string `json:"name"`
}

type assigneeNode struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type labelsConn struct {
	Nodes []labelNode `json:"nodes"`
}

type labelNode struct {
	Name string `json:"name"`
}

type relationsConn struct {
	Nodes []relationNode `json:"nodes"`
}

type relationNode struct {
	Type         string          `json:"type"`
	RelatedIssue relatedIssueNode `json:"relatedIssue"`
}

type relatedIssueNode struct {
	ID         string    `json:"id"`
	Identifier string    `json:"identifier"`
	State      stateNode `json:"state"`
}

// --- Single Issue Response ---

type singleIssueData struct {
	Issue issueNode `json:"issue"`
}

// --- Nodes (batch) Response ---

type nodesData struct {
	Nodes []nodeItem `json:"nodes"`
}

type nodeItem struct {
	ID    string     `json:"id"`
	State *stateNode `json:"state,omitempty"`
}

// --- User Response ---

type usersData struct {
	Users usersConnection `json:"users"`
}

type usersConnection struct {
	Nodes []userNode `json:"nodes"`
}

type userNode struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// --- Mutation Responses ---

type assignIssueData struct {
	IssueUpdate mutationResult `json:"issueUpdate"`
}

type unassignIssueData struct {
	IssueUpdate mutationResult `json:"issueUpdate"`
}

type mutationResult struct {
	Success bool       `json:"success"`
	Issue   *issueNode `json:"issue,omitempty"`
}
