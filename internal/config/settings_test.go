package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestEnsureSettingsFileCreatesDefaults verifies missing settings are created with defaults.
func TestEnsureSettingsFileCreatesDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "nested", "config.json")

	settings, err := EnsureSettingsFile(settingsPath)
	if err != nil {
		t.Fatalf("EnsureSettingsFile() error: %v", err)
	}

	if _, err := os.Stat(settingsPath); err != nil {
		t.Fatalf("settings file not created: %v", err)
	}

	if _, err := os.Stat(filepath.Dir(settingsPath)); err != nil {
		t.Fatalf("settings directory not created: %v", err)
	}

	assertSettingsEqual(t, settings, DefaultSettings())
}

// TestLoadSettingsAppliesDefaults verifies missing fields use default values.
func TestLoadSettingsAppliesDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "config.json")

	data := []byte(`{"page_size":123}`)
	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		t.Fatalf("write settings file: %v", err)
	}

	settings, err := LoadSettings(settingsPath)
	if err != nil {
		t.Fatalf("LoadSettings() error: %v", err)
	}

	expected := DefaultSettings()
	expected.PageSize = 123
	assertSettingsEqual(t, settings, expected)
}

// TestLoadSettingsPreservesEmptyLogFile ensures an empty log file disables logging.
func TestLoadSettingsPreservesEmptyLogFile(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "config.json")

	data := []byte(`{"log_file": ""}`)
	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		t.Fatalf("write settings file: %v", err)
	}

	settings, err := LoadSettings(settingsPath)
	if err != nil {
		t.Fatalf("LoadSettings() error: %v", err)
	}

	expected := DefaultSettings()
	expected.LogFile = ""
	assertSettingsEqual(t, settings, expected)
}

// TestConfigFromSettingsValidation checks invalid settings are rejected.
func TestConfigFromSettingsValidation(t *testing.T) {
	base := DefaultSettings()

	tests := []struct {
		name   string
		mutate func(Settings) Settings
	}{
		{
			name: "invalid timeout",
			mutate: func(settings Settings) Settings {
				settings.Timeout = "not-a-duration"
				return settings
			},
		},
		{
			name: "invalid cache ttl",
			mutate: func(settings Settings) Settings {
				settings.CacheTTL = "bad-duration"
				return settings
			},
		},
		{
			name: "page size too low",
			mutate: func(settings Settings) Settings {
				settings.PageSize = 0
				return settings
			},
		},
		{
			name: "page size too high",
			mutate: func(settings Settings) Settings {
				settings.PageSize = 300
				return settings
			},
		},
		{
			name: "invalid log level",
			mutate: func(settings Settings) Settings {
				settings.LogLevel = "verbose"
				return settings
			},
		},
		{
			name: "invalid theme",
			mutate: func(settings Settings) Settings {
				settings.Theme = "rainbow"
				return settings
			},
		},
		{
			name: "invalid density",
			mutate: func(settings Settings) Settings {
				settings.Density = "ultra"
				return settings
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := tt.mutate(base)
			_, err := ConfigFromSettings("test-key", settings)
			if err == nil {
				t.Errorf("ConfigFromSettings() expected error for %s", tt.name)
			}
		})
	}
}

// TestConfigFromSettingsRequiresAPIKey verifies API key is mandatory.
func TestConfigFromSettingsRequiresAPIKey(t *testing.T) {
	_, err := ConfigFromSettings("", DefaultSettings())
	if err == nil {
		t.Error("ConfigFromSettings() expected error when API key is empty")
	}
}

// TestDefaultSettingsAgentDefaults verifies agent command defaults are set.
func TestDefaultSettingsAgentDefaults(t *testing.T) {
	settings := DefaultSettings()
	if len(settings.AgentCommands) != 2 {
		t.Errorf("AgentCommands length = %d, want 2", len(settings.AgentCommands))
	}
	if settings.AgentCommands[0].Name != "Claude" {
		t.Errorf("AgentCommands[0].Name = %q, want %q", settings.AgentCommands[0].Name, "Claude")
	}
	if settings.AgentCommands[0].Command != "claude {prompt}" {
		t.Errorf("AgentCommands[0].Command = %q, want %q", settings.AgentCommands[0].Command, "claude {prompt}")
	}
	if settings.AgentWorkspace != "" {
		t.Errorf("AgentWorkspace = %q, want empty string", settings.AgentWorkspace)
	}
	if settings.Theme != DefaultTheme {
		t.Errorf("Theme = %q, want %q", settings.Theme, DefaultTheme)
	}
	if settings.Density != DefaultDensity {
		t.Errorf("Density = %q, want %q", settings.Density, DefaultDensity)
	}
}

// TestMigrateAgentCommands verifies backward compatibility migration.
func TestMigrateAgentCommands(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		model    string
		sandbox  string
		wantName string
		wantCmd  string
	}{
		{
			name:     "empty provider uses defaults",
			provider: "",
			wantName: "Claude",
			wantCmd:  "claude {prompt}",
		},
		{
			name:     "unknown provider uses defaults",
			provider: "unknown",
			wantName: "Claude",
			wantCmd:  "claude {prompt}",
		},
		{
			name:     "claude with model and skip permissions",
			provider: "claude",
			model:    "opus",
			sandbox:  "dangerously-skip-permissions",
			wantName: "Claude (migrated)",
			wantCmd:  "claude --model opus --dangerously-skip-permissions {prompt}",
		},
		{
			name:     "claude with no options",
			provider: "claude",
			wantName: "Claude (migrated)",
			wantCmd:  "claude {prompt}",
		},
		{
			name:     "cursor with sandbox and model",
			provider: "cursor",
			sandbox:  "enabled",
			model:    "gpt-5.2",
			wantName: "Cursor (migrated)",
			wantCmd:  "cursor-agent --sandbox enabled --model gpt-5.2 {prompt}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmds := migrateAgentCommands(tt.provider, tt.model, tt.sandbox)
			if tt.provider == "" || tt.provider == "unknown" {
				// Should return default commands
				if len(cmds) != 2 {
					t.Fatalf("expected 2 default commands, got %d", len(cmds))
				}
				if cmds[0].Name != tt.wantName {
					t.Errorf("Name = %q, want %q", cmds[0].Name, tt.wantName)
				}
				if cmds[0].Command != tt.wantCmd {
					t.Errorf("Command = %q, want %q", cmds[0].Command, tt.wantCmd)
				}
				return
			}
			if len(cmds) != 1 {
				t.Fatalf("expected 1 migrated command, got %d", len(cmds))
			}
			if cmds[0].Name != tt.wantName {
				t.Errorf("Name = %q, want %q", cmds[0].Name, tt.wantName)
			}
			if cmds[0].Command != tt.wantCmd {
				t.Errorf("Command = %q, want %q", cmds[0].Command, tt.wantCmd)
			}
		})
	}
}

// TestLoadSettingsWithAgentCommands verifies agent_commands are loaded from file.
func TestLoadSettingsWithAgentCommands(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "config.json")

	data := []byte(`{"agent_commands": [{"name": "Custom", "command": "my-agent {prompt}"}]}`)
	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		t.Fatalf("write settings file: %v", err)
	}

	settings, err := LoadSettings(settingsPath)
	if err != nil {
		t.Fatalf("LoadSettings() error: %v", err)
	}

	if len(settings.AgentCommands) != 1 {
		t.Fatalf("AgentCommands length = %d, want 1", len(settings.AgentCommands))
	}
	if settings.AgentCommands[0].Name != "Custom" {
		t.Errorf("AgentCommands[0].Name = %q, want %q", settings.AgentCommands[0].Name, "Custom")
	}
	if settings.AgentCommands[0].Command != "my-agent {prompt}" {
		t.Errorf("AgentCommands[0].Command = %q, want %q", settings.AgentCommands[0].Command, "my-agent {prompt}")
	}
}

// TestLoadSettingsLegacyMigration verifies legacy fields are migrated when agent_commands is absent.
func TestLoadSettingsLegacyMigration(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "config.json")

	data := []byte(`{"agent_provider": "claude", "agent_model": "opus", "agent_sandbox": "dangerously-skip-permissions"}`)
	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		t.Fatalf("write settings file: %v", err)
	}

	settings, err := LoadSettings(settingsPath)
	if err != nil {
		t.Fatalf("LoadSettings() error: %v", err)
	}

	if len(settings.AgentCommands) != 1 {
		t.Fatalf("AgentCommands length = %d, want 1", len(settings.AgentCommands))
	}
	if settings.AgentCommands[0].Name != "Claude (migrated)" {
		t.Errorf("Name = %q, want %q", settings.AgentCommands[0].Name, "Claude (migrated)")
	}
	if settings.AgentCommands[0].Command != "claude --model opus --dangerously-skip-permissions {prompt}" {
		t.Errorf("Command = %q, want %q", settings.AgentCommands[0].Command, "claude --model opus --dangerously-skip-permissions {prompt}")
	}
}

// assertSettingsEqual compares settings values in tests.
func assertSettingsEqual(t *testing.T, got Settings, want Settings) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Settings mismatch: got %+v, want %+v", got, want)
	}
}
