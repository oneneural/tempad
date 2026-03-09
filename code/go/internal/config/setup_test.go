package config

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAPIKey_Empty(t *testing.T) {
	err := ValidateAPIKey(context.Background(), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestValidateAPIKey_Whitespace(t *testing.T) {
	err := ValidateAPIKey(context.Background(), "   ")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestValidateIdentity_Empty(t *testing.T) {
	err := ValidateIdentity(context.Background(), "fake-key", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestValidateIdentity_Whitespace(t *testing.T) {
	err := ValidateIdentity(context.Background(), "fake-key", "   ")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestLookupBinary_Empty(t *testing.T) {
	_, err := LookupBinary("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestLookupBinary_NotFound(t *testing.T) {
	_, err := LookupBinary("this-binary-definitely-does-not-exist-xyz123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found on PATH")
}

func TestLookupBinary_Found(t *testing.T) {
	// "go" should always be available in the test environment.
	path, err := LookupBinary("go")
	assert.NoError(t, err)
	assert.NotEmpty(t, path)
}

func TestDefaultWorkspaceRoot(t *testing.T) {
	root := DefaultWorkspaceRoot()
	assert.NotEmpty(t, root)
	assert.Contains(t, root, "tempad")
}

func TestEnsureDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "test-setup", "nested")
	err := EnsureDirectory(dir)
	require.NoError(t, err)

	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestWriteUserConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &UserConfig{
		Tracker: UserTrackerConfig{
			Identity: "test@example.com",
			APIKey:   "lin_api_test123",
		},
		IDE: UserIDEConfig{
			Command: "code",
		},
		Agent: UserAgentConfig{
			Command: "claude",
		},
		Display: UserDisplayConfig{
			Theme: "auto",
		},
		Logging: UserLoggingConfig{
			Level: "info",
		},
	}

	err := WriteUserConfig(path, cfg)
	require.NoError(t, err)

	// Verify the file was written.
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)

	assert.Contains(t, content, "test@example.com")
	assert.Contains(t, content, "lin_api_test123")
	assert.Contains(t, content, "code")
	assert.Contains(t, content, "claude")

	// Verify it can be loaded back.
	loaded, err := LoadUserConfig(path)
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", loaded.Tracker.Identity)
	assert.Equal(t, "lin_api_test123", loaded.Tracker.APIKey)
	assert.Equal(t, "code", loaded.IDE.Command)
	assert.Equal(t, "claude", loaded.Agent.Command)
}

func TestWriteUserConfig_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "deep")
	path := filepath.Join(dir, "config.yaml")

	cfg := &UserConfig{
		Tracker: UserTrackerConfig{Identity: "x@y.com"},
	}

	err := WriteUserConfig(path, cfg)
	require.NoError(t, err)

	_, err = os.Stat(path)
	assert.NoError(t, err)
}

func TestKnownIDEs_NonEmpty(t *testing.T) {
	assert.NotEmpty(t, KnownIDEs)
	for _, ide := range KnownIDEs {
		assert.NotEmpty(t, ide.Name)
		assert.NotEmpty(t, ide.Command)
	}
}

func TestKnownAgents_NonEmpty(t *testing.T) {
	assert.NotEmpty(t, KnownAgents)
	for _, agent := range KnownAgents {
		assert.NotEmpty(t, agent.Name)
		assert.NotEmpty(t, agent.Command)
	}
}

// TestValidateAPIKey_MockServer tests API key validation against a mock server.
func TestValidateAPIKey_MockServer(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   any
		wantErr    string
	}{
		{
			name:       "valid key",
			statusCode: http.StatusOK,
			response: map[string]any{
				"data": map[string]any{
					"viewer": map[string]any{"id": "user-123"},
				},
			},
			wantErr: "",
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			response:   map[string]string{"error": "unauthorized"},
			wantErr:    "API key invalid",
		},
		{
			name:       "forbidden",
			statusCode: http.StatusForbidden,
			response:   map[string]string{"error": "forbidden"},
			wantErr:    "API key invalid",
		},
		{
			name:       "graphql error",
			statusCode: http.StatusOK,
			response: map[string]any{
				"errors": []map[string]string{
					{"message": "Authentication required"},
				},
			},
			wantErr: "API key invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.response)
			}))
			defer srv.Close()

			err := validateAPIKeyWithEndpoint(context.Background(), "test-key", srv.URL)
			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateIdentity_MockServer tests identity validation against a mock server.
func TestValidateIdentity_MockServer(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   any
		wantErr    string
	}{
		{
			name:       "user found",
			statusCode: http.StatusOK,
			response: map[string]any{
				"data": map[string]any{
					"users": map[string]any{
						"nodes": []map[string]string{
							{"id": "user-1", "email": "test@example.com"},
						},
					},
				},
			},
			wantErr: "",
		},
		{
			name:       "user not found",
			statusCode: http.StatusOK,
			response: map[string]any{
				"data": map[string]any{
					"users": map[string]any{
						"nodes": []any{},
					},
				},
			},
			wantErr: "no Linear user found",
		},
		{
			name:       "api error",
			statusCode: http.StatusInternalServerError,
			response:   "error",
			wantErr:    "API request failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.response)
			}))
			defer srv.Close()

			err := validateIdentityWithEndpoint(context.Background(), "test-key", "test@example.com", srv.URL)
			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
