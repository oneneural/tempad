package domain

// Workspace represents a filesystem workspace assigned to one issue identifier.
// See Spec Section 4.1.5.
type Workspace struct {
	// Path is the absolute workspace directory path.
	Path string `json:"path"`

	// WorkspaceKey is the sanitized issue identifier used as the directory name.
	WorkspaceKey string `json:"workspace_key"`

	// CreatedNow indicates whether the workspace was newly created (true)
	// or reused from a previous run (false). Used to gate the after_create hook.
	CreatedNow bool `json:"created_now"`
}
