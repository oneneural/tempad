package tracker

import (
	"errors"
	"fmt"
	"testing"
)

func TestUnsupportedTrackerKindError(t *testing.T) {
	err := &UnsupportedTrackerKindError{Kind: "jira"}
	if err.Error() != `unsupported tracker kind: "jira"` {
		t.Errorf("unexpected message: %s", err.Error())
	}

	var target *UnsupportedTrackerKindError
	if !errors.As(err, &target) {
		t.Error("errors.As should match UnsupportedTrackerKindError")
	}
	if target.Kind != "jira" {
		t.Errorf("expected Kind=jira, got %s", target.Kind)
	}
}

func TestMissingTrackerAPIKeyError(t *testing.T) {
	err := &MissingTrackerAPIKeyError{}
	if err.Error() != "tracker API key is missing" {
		t.Errorf("unexpected message: %s", err.Error())
	}

	var target *MissingTrackerAPIKeyError
	if !errors.As(err, &target) {
		t.Error("errors.As should match MissingTrackerAPIKeyError")
	}
}

func TestMissingTrackerProjectSlugError(t *testing.T) {
	err := &MissingTrackerProjectSlugError{}
	if err.Error() != "tracker project slug is missing" {
		t.Errorf("unexpected message: %s", err.Error())
	}

	var target *MissingTrackerProjectSlugError
	if !errors.As(err, &target) {
		t.Error("errors.As should match MissingTrackerProjectSlugError")
	}
}

func TestMissingTrackerIdentityError(t *testing.T) {
	err := &MissingTrackerIdentityError{}
	if err.Error() != "tracker identity is missing" {
		t.Errorf("unexpected message: %s", err.Error())
	}

	var target *MissingTrackerIdentityError
	if !errors.As(err, &target) {
		t.Error("errors.As should match MissingTrackerIdentityError")
	}
}

func TestAPIRequestError(t *testing.T) {
	cause := fmt.Errorf("connection refused")
	err := &APIRequestError{Message: "POST /graphql", Cause: cause}

	if err.Error() != "tracker API request failed: POST /graphql: connection refused" {
		t.Errorf("unexpected message: %s", err.Error())
	}

	var target *APIRequestError
	if !errors.As(err, &target) {
		t.Error("errors.As should match APIRequestError")
	}
	if target.Message != "POST /graphql" {
		t.Errorf("expected Message='POST /graphql', got %s", target.Message)
	}

	// Unwrap should return the cause.
	if !errors.Is(err, cause) {
		t.Error("errors.Is should match the wrapped cause")
	}
}

func TestAPIStatusError(t *testing.T) {
	err := &APIStatusError{StatusCode: 401, Body: "Unauthorized"}
	if err.Error() != "tracker API returned status 401: Unauthorized" {
		t.Errorf("unexpected message: %s", err.Error())
	}

	var target *APIStatusError
	if !errors.As(err, &target) {
		t.Error("errors.As should match APIStatusError")
	}
	if target.StatusCode != 401 {
		t.Errorf("expected StatusCode=401, got %d", target.StatusCode)
	}
}

func TestAPIErrorsError(t *testing.T) {
	err := &APIErrorsError{Errors: []string{"field not found", "invalid query"}}
	msg := err.Error()
	if msg != "tracker API GraphQL errors: [field not found invalid query]" {
		t.Errorf("unexpected message: %s", msg)
	}

	var target *APIErrorsError
	if !errors.As(err, &target) {
		t.Error("errors.As should match APIErrorsError")
	}
	if len(target.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(target.Errors))
	}
}

func TestRateLimitError(t *testing.T) {
	err := &RateLimitError{RetryAfterSecs: 60}
	if err.Error() != "tracker API rate limit exceeded, retry after 60 seconds" {
		t.Errorf("unexpected message: %s", err.Error())
	}

	var target *RateLimitError
	if !errors.As(err, &target) {
		t.Error("errors.As should match RateLimitError")
	}
	if target.RetryAfterSecs != 60 {
		t.Errorf("expected RetryAfterSecs=60, got %d", target.RetryAfterSecs)
	}
}

func TestClaimConflictError(t *testing.T) {
	err := &ClaimConflictError{
		IssueID:        "abc-123",
		ExpectedUser:   "me@example.com",
		ActualAssignee: "other@example.com",
	}
	expected := `claim conflict for issue abc-123: expected assignee "me@example.com", got "other@example.com"`
	if err.Error() != expected {
		t.Errorf("unexpected message: %s", err.Error())
	}

	var target *ClaimConflictError
	if !errors.As(err, &target) {
		t.Error("errors.As should match ClaimConflictError")
	}
	if target.IssueID != "abc-123" {
		t.Errorf("expected IssueID=abc-123, got %s", target.IssueID)
	}
}

func TestWrappedErrorChain(t *testing.T) {
	inner := fmt.Errorf("dns resolution failed")
	apiErr := &APIRequestError{Message: "POST", Cause: inner}
	wrapped := fmt.Errorf("tracker operation failed: %w", apiErr)

	var target *APIRequestError
	if !errors.As(wrapped, &target) {
		t.Error("errors.As should find APIRequestError through wrapping")
	}
	if !errors.Is(wrapped, inner) {
		t.Error("errors.Is should find inner error through chain")
	}
}
