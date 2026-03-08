package config

import (
	"fmt"
	"strings"
)

// ValidationError represents a configuration validation failure.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("config validation: %s — %s", e.Field, e.Message)
}

// ValidationErrors collects multiple validation failures.
type ValidationErrors struct {
	Errors []*ValidationError
}

func (e *ValidationErrors) Error() string {
	if len(e.Errors) == 0 {
		return "config validation: unknown error"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	msgs := make([]string, len(e.Errors))
	for i, err := range e.Errors {
		msgs[i] = err.Error()
	}
	return fmt.Sprintf("config validation failed (%d errors):\n  %s",
		len(e.Errors), strings.Join(msgs, "\n  "))
}

// ValidateForStartup checks that the configuration is valid for starting
// TEMPAD. Fails startup with a clear error if invalid.
// See Spec Section 8.3.
func ValidateForStartup(cfg *ServiceConfig, mode string) error {
	var errs []*ValidationError

	// tracker.kind is present and supported.
	if cfg.TrackerKind == "" {
		errs = append(errs, &ValidationError{
			Field:   "tracker.kind",
			Message: "tracker kind is required (set in WORKFLOW.md front matter)",
		})
	} else if cfg.TrackerKind != "linear" {
		errs = append(errs, &ValidationError{
			Field:   "tracker.kind",
			Message: fmt.Sprintf("unsupported tracker kind %q (only \"linear\" is supported)", cfg.TrackerKind),
		})
	}

	// tracker.api_key is present after $VAR resolution.
	if cfg.TrackerAPIKey == "" {
		errs = append(errs, &ValidationError{
			Field:   "tracker.api_key",
			Message: "tracker API key is required (set in WORKFLOW.md or ~/.tempad/config.yaml; if using $VAR, ensure the environment variable is set)",
		})
	}

	// tracker.project_slug is present when kind=linear.
	if cfg.TrackerKind == "linear" && cfg.TrackerProjectSlug == "" {
		errs = append(errs, &ValidationError{
			Field:   "tracker.project_slug",
			Message: "project slug is required for Linear tracker (set in WORKFLOW.md front matter)",
		})
	}

	// tracker.identity is present.
	if cfg.TrackerIdentity == "" {
		errs = append(errs, &ValidationError{
			Field:   "tracker.identity",
			Message: "tracker identity is required (set in ~/.tempad/config.yaml or use --identity flag)",
		})
	}

	// agent.command is present for daemon mode.
	if mode == "daemon" && cfg.AgentCommand == "" {
		errs = append(errs, &ValidationError{
			Field:   "agent.command",
			Message: "agent command is required for daemon mode (set in WORKFLOW.md or ~/.tempad/config.yaml)",
		})
	}

	if len(errs) > 0 {
		return &ValidationErrors{Errors: errs}
	}
	return nil
}

// ValidateForDispatch checks that the configuration is valid for dispatching
// work in the current tick. Same checks as startup, called per-tick in daemon mode.
// See Spec Section 8.3.
func ValidateForDispatch(cfg *ServiceConfig, mode string) error {
	return ValidateForStartup(cfg, mode)
}
