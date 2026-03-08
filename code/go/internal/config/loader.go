package config

// Merge combines configuration from all sources according to the merge
// precedence rules in Spec Section 8.1:
//
//	CLI flags > User config > Repo config (WORKFLOW.md) > Env vars > Defaults
//
// Personal fields (identity, api_key, ide, agent command) → user config wins.
// Team fields (hooks, states, workspace root, concurrency) → repo config wins.
func Merge(cli *CLIFlags, user *UserConfig, workflow *WorkflowDefinition) *ServiceConfig {
	cfg := Defaults()

	if workflow != nil {
		cfg.WorkflowPath = "" // caller sets this
		applyWorkflowConfig(cfg, workflow.Config)
	}

	if user != nil {
		applyUserConfig(cfg, user)
	}

	if cli != nil {
		applyCLIFlags(cfg, cli)
	}

	// Resolve $VAR references after merge.
	cfg.TrackerAPIKey = ResolveEnvVar(cfg.TrackerAPIKey)
	cfg.WorkspaceRoot = ExpandHome(cfg.WorkspaceRoot)
	cfg.LogFile = ExpandHome(cfg.LogFile)

	return cfg
}

// applyWorkflowConfig applies repo-level settings from WORKFLOW.md front matter.
// Repo config wins for team-shared settings.
func applyWorkflowConfig(cfg *ServiceConfig, m map[string]any) {
	if m == nil {
		return
	}

	// Tracker settings (team-owned except identity/api_key).
	if v, ok := getNestedString(m, "tracker.kind"); ok {
		cfg.TrackerKind = v
	}
	if v, ok := getNestedString(m, "tracker.endpoint"); ok {
		cfg.TrackerEndpoint = v
	}
	if v, ok := getNestedString(m, "tracker.api_key"); ok {
		cfg.TrackerAPIKey = v
	}
	if v, ok := getNestedString(m, "tracker.project_slug"); ok {
		cfg.TrackerProjectSlug = v
	}
	if v, ok := getNestedStringList(m, "tracker.active_states"); ok {
		cfg.ActiveStates = v
	}
	if v, ok := getNestedStringList(m, "tracker.terminal_states"); ok {
		cfg.TerminalStates = v
	}

	// Polling (team-owned).
	if v, ok := getNestedInt(m, "polling.interval_ms"); ok {
		cfg.PollIntervalMs = v
	}

	// Workspace (team-owned).
	if v, ok := getNestedString(m, "workspace.root"); ok {
		cfg.WorkspaceRoot = v
	}

	// Hooks (team-owned, not overridable by user config).
	if v, ok := getNestedString(m, "hooks.after_create"); ok {
		cfg.AfterCreateHook = v
	}
	if v, ok := getNestedString(m, "hooks.before_run"); ok {
		cfg.BeforeRunHook = v
	}
	if v, ok := getNestedString(m, "hooks.after_run"); ok {
		cfg.AfterRunHook = v
	}
	if v, ok := getNestedString(m, "hooks.before_remove"); ok {
		cfg.BeforeRemoveHook = v
	}
	if v, ok := getNestedInt(m, "hooks.timeout_ms"); ok {
		cfg.HookTimeoutMs = v
	}

	// Agent (team-owned for most fields).
	if v, ok := getNestedString(m, "agent.command"); ok {
		cfg.AgentCommand = v
	}
	if v, ok := getNestedString(m, "agent.args"); ok {
		cfg.AgentArgs = v
	}
	if v, ok := getNestedString(m, "agent.prompt_delivery"); ok {
		cfg.PromptDelivery = v
	}
	if v, ok := getNestedInt(m, "agent.max_concurrent"); ok {
		cfg.MaxConcurrent = v
	}
	if v, ok := getNestedIntMap(m, "agent.max_concurrent_by_state"); ok {
		cfg.MaxConcurrentByState = v
	}
	if v, ok := getNestedInt(m, "agent.max_turns"); ok {
		cfg.MaxTurns = v
	}
	if v, ok := getNestedInt(m, "agent.max_retries"); ok {
		cfg.MaxRetries = v
	}
	if v, ok := getNestedInt(m, "agent.max_retry_backoff_ms"); ok {
		cfg.MaxRetryBackoffMs = v
	}
	if v, ok := getNestedInt(m, "agent.turn_timeout_ms"); ok {
		cfg.TurnTimeoutMs = v
	}
	if v, ok := getNestedInt(m, "agent.stall_timeout_ms"); ok {
		cfg.StallTimeoutMs = v
	}
	if v, ok := getNestedInt(m, "agent.read_timeout_ms"); ok {
		cfg.ReadTimeoutMs = v
	}
}

// applyUserConfig applies personal preferences from ~/.tempad/config.yaml.
// User config wins for personal fields; does NOT override team settings
// like hooks, states, or concurrency.
func applyUserConfig(cfg *ServiceConfig, user *UserConfig) {
	// Personal: tracker identity and api_key (user wins).
	if user.Tracker.Identity != "" {
		cfg.TrackerIdentity = user.Tracker.Identity
	}
	if user.Tracker.APIKey != "" {
		cfg.TrackerAPIKey = user.Tracker.APIKey
	}

	// Personal: IDE preferences (user wins).
	if user.IDE.Command != "" {
		cfg.IDECommand = user.IDE.Command
	}
	if user.IDE.Args != "" {
		cfg.IDEArgs = user.IDE.Args
	}

	// Personal: agent command (user wins over repo).
	if user.Agent.Command != "" {
		cfg.AgentCommand = user.Agent.Command
	}
	if user.Agent.Args != "" {
		cfg.AgentArgs = user.Agent.Args
	}

	// Personal: display theme.
	if user.Display.Theme != "" {
		cfg.Theme = user.Display.Theme
	}

	// Personal: logging preferences.
	if user.Logging.Level != "" {
		cfg.LogLevel = user.Logging.Level
	}
	if user.Logging.File != "" {
		cfg.LogFile = user.Logging.File
	}
}

// applyCLIFlags applies command-line flags (highest precedence).
func applyCLIFlags(cfg *ServiceConfig, cli *CLIFlags) {
	if cli.Identity != "" {
		cfg.TrackerIdentity = cli.Identity
	}
	if cli.Agent != "" {
		cfg.AgentCommand = cli.Agent
	}
	if cli.IDE != "" {
		cfg.IDECommand = cli.IDE
	}
	if cli.Port > 0 {
		cfg.ServerPort = cli.Port
	}
	if cli.LogLevel != "" {
		cfg.LogLevel = cli.LogLevel
	}
	if cli.WorkflowPath != "" {
		cfg.WorkflowPath = cli.WorkflowPath
	}
}

// Load performs the full config loading pipeline:
// 1. Load workflow file.
// 2. Load user config.
// 3. Merge all sources.
// 4. Resolve $VAR references.
//
// Returns the merged config and the parsed workflow (for prompt template access).
func Load(cli *CLIFlags) (*ServiceConfig, *WorkflowDefinition, error) {
	workflowPath := ""
	if cli != nil {
		workflowPath = cli.WorkflowPath
	}

	workflow, err := LoadWorkflow(workflowPath)
	if err != nil {
		return nil, nil, err
	}

	userCfg, err := LoadUserConfig("")
	if err != nil {
		return nil, nil, err
	}

	cfg := Merge(cli, userCfg, workflow)
	if workflowPath != "" {
		cfg.WorkflowPath = workflowPath
	} else {
		cfg.WorkflowPath = "WORKFLOW.md"
	}

	// Store the prompt template for IDE agent context.
	if workflow != nil {
		cfg.PromptTemplate = workflow.PromptTemplate
	}

	return cfg, workflow, nil
}
