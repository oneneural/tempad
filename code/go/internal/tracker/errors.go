package tracker

import "fmt"

// Error types for tracker operations per Spec Section 13.4.

// UnsupportedTrackerKindError indicates an unknown tracker kind.
type UnsupportedTrackerKindError struct {
	Kind string
}

func (e *UnsupportedTrackerKindError) Error() string {
	return fmt.Sprintf("unsupported tracker kind: %q", e.Kind)
}

// MissingTrackerAPIKeyError indicates no API key was configured.
type MissingTrackerAPIKeyError struct{}

func (e *MissingTrackerAPIKeyError) Error() string {
	return "tracker API key is missing"
}

// MissingTrackerProjectSlugError indicates no project slug was configured.
type MissingTrackerProjectSlugError struct{}

func (e *MissingTrackerProjectSlugError) Error() string {
	return "tracker project slug is missing"
}

// MissingTrackerIdentityError indicates no identity was configured.
type MissingTrackerIdentityError struct{}

func (e *MissingTrackerIdentityError) Error() string {
	return "tracker identity is missing"
}

// APIRequestError wraps transport-level errors.
type APIRequestError struct {
	Message string
	Cause   error
}

func (e *APIRequestError) Error() string {
	return fmt.Sprintf("tracker API request failed: %s: %v", e.Message, e.Cause)
}

func (e *APIRequestError) Unwrap() error {
	return e.Cause
}

// APIStatusError wraps non-200 HTTP responses.
type APIStatusError struct {
	StatusCode int
	Body       string
}

func (e *APIStatusError) Error() string {
	return fmt.Sprintf("tracker API returned status %d: %s", e.StatusCode, e.Body)
}

// APIErrorsError wraps GraphQL-level errors.
type APIErrorsError struct {
	Errors []string
}

func (e *APIErrorsError) Error() string {
	return fmt.Sprintf("tracker API GraphQL errors: %v", e.Errors)
}

// RateLimitError indicates the API rate limit has been exceeded.
type RateLimitError struct {
	RetryAfterSecs int
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("tracker API rate limit exceeded, retry after %d seconds", e.RetryAfterSecs)
}

// ClaimConflictError indicates the issue was claimed by someone else.
type ClaimConflictError struct {
	IssueID        string
	ExpectedUser   string
	ActualAssignee string
}

func (e *ClaimConflictError) Error() string {
	return fmt.Sprintf("claim conflict for issue %s: expected assignee %q, got %q",
		e.IssueID, e.ExpectedUser, e.ActualAssignee)
}
