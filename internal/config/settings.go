package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SettingsFile represents the on-disk JSON with optional fields.
type SettingsFile struct {
	APIEndpoint    *string         `json:"api_endpoint"`
	Timeout        *string         `json:"timeout"`
	PageSize       *int            `json:"page_size"`
	CacheTTL       *string         `json:"cache_ttl"`
	LogFile        *string         `json:"log_file"`
	LogLevel       *string         `json:"log_level"`
	Theme          *string         `json:"theme"`
	Density        *string         `json:"density"`
	AgentCommands  *[]AgentCommand `json:"agent_commands"`
	AgentWorkspace *string         `json:"agent_workspace"`
	// Legacy fields (read-only for migration)
	AgentProvider *string `json:"agent_provider"`
	AgentSandbox  *string `json:"agent_sandbox"`
	AgentModel    *string `json:"agent_model"`
}

// Settings contains concrete settings values for UI and persistence.
type Settings struct {
	APIEndpoint    string         `json:"api_endpoint"`
	Timeout        string         `json:"timeout"`
	PageSize       int            `json:"page_size"`
	CacheTTL       string         `json:"cache_ttl"`
	LogFile        string         `json:"log_file"`
	LogLevel       string         `json:"log_level"`
	Theme          string         `json:"theme"`
	Density        string         `json:"density"`
	AgentCommands  []AgentCommand `json:"agent_commands"`
	AgentWorkspace string         `json:"agent_workspace"`
}

// DefaultSettings returns the default settings for the config file and UI.
func DefaultSettings() Settings {
	return Settings{
		APIEndpoint:    DefaultAPIEndpoint,
		Timeout:        DefaultTimeout.String(),
		PageSize:       DefaultPageSize,
		CacheTTL:       DefaultCacheTTL.String(),
		LogFile:        getDefaultLogFile(),
		LogLevel:       DefaultLogLevel,
		Theme:          DefaultTheme,
		Density:        DefaultDensity,
		AgentCommands:  DefaultAgentCommands(),
		AgentWorkspace: "",
	}
}

// SettingsFromConfig converts runtime config into settings values.
func SettingsFromConfig(cfg Config) Settings {
	return Settings{
		APIEndpoint:    cfg.APIEndpoint,
		Timeout:        cfg.Timeout.String(),
		PageSize:       cfg.PageSize,
		CacheTTL:       cfg.CacheTTL.String(),
		LogFile:        cfg.LogFile,
		LogLevel:       cfg.LogLevel,
		Theme:          cfg.Theme,
		Density:        cfg.Density,
		AgentCommands:  cfg.AgentCommands,
		AgentWorkspace: cfg.AgentWorkspace,
	}
}

// ConfigFromSettings builds runtime configuration from settings and API key.
func ConfigFromSettings(apiKey string, settings Settings) (Config, error) {
	if apiKey == "" {
		return Config{}, fmt.Errorf("%s environment variable is not set", LinearAPIKeyEnv)
	}

	timeout, err := parseDuration(settings.Timeout, "timeout")
	if err != nil {
		return Config{}, err
	}

	cacheTTL, err := parseDuration(settings.CacheTTL, "cache_ttl")
	if err != nil {
		return Config{}, err
	}

	if err := validatePageSize(settings.PageSize, "page_size"); err != nil {
		return Config{}, err
	}

	if err := validateLogLevel(settings.LogLevel, "log_level"); err != nil {
		return Config{}, err
	}

	theme := strings.TrimSpace(settings.Theme)
	if theme == "" {
		theme = DefaultTheme
	}
	if err := validateTheme(theme, "theme"); err != nil {
		return Config{}, err
	}

	density := strings.TrimSpace(settings.Density)
	if density == "" {
		density = DefaultDensity
	}
	if err := validateDensity(density, "density"); err != nil {
		return Config{}, err
	}

	agentCommands := settings.AgentCommands
	if agentCommands == nil {
		agentCommands = DefaultAgentCommands()
	}

	return Config{
		LinearAPIKey:   apiKey,
		APIEndpoint:    settings.APIEndpoint,
		Timeout:        timeout,
		PageSize:       settings.PageSize,
		CacheTTL:       cacheTTL,
		LogFile:        settings.LogFile,
		LogLevel:       settings.LogLevel,
		Theme:          theme,
		Density:        density,
		AgentCommands:  agentCommands,
		AgentWorkspace: settings.AgentWorkspace,
	}, nil
}

// ConfigFilePath returns the default settings file path.
func ConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}

	return filepath.Join(homeDir, ".linear-tui", "config.json"), nil
}

// EnsureSettingsFile ensures the settings file exists and returns its settings.
func EnsureSettingsFile(path string) (Settings, error) {
	if path == "" {
		return Settings{}, fmt.Errorf("settings path is empty")
	}

	if _, err := os.Stat(path); err == nil {
		return LoadSettings(path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Settings{}, fmt.Errorf("stat settings file: %w", err)
	}

	settings := DefaultSettings()
	if err := SaveSettings(path, settings); err != nil {
		return Settings{}, err
	}

	return settings, nil
}

// LoadSettings loads settings from a JSON file and applies defaults.
func LoadSettings(path string) (Settings, error) {
	if path == "" {
		return Settings{}, fmt.Errorf("settings path is empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Settings{}, fmt.Errorf("read settings file: %w", err)
	}

	var file SettingsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return Settings{}, fmt.Errorf("parse settings file: %w", err)
	}

	settings := DefaultSettings()
	if file.APIEndpoint != nil {
		settings.APIEndpoint = *file.APIEndpoint
	}
	if file.Timeout != nil {
		settings.Timeout = *file.Timeout
	}
	if file.PageSize != nil {
		settings.PageSize = *file.PageSize
	}
	if file.CacheTTL != nil {
		settings.CacheTTL = *file.CacheTTL
	}
	if file.LogFile != nil {
		settings.LogFile = *file.LogFile
	}
	if file.LogLevel != nil {
		settings.LogLevel = *file.LogLevel
	}
	if file.Theme != nil {
		settings.Theme = *file.Theme
	}
	if file.Density != nil {
		settings.Density = *file.Density
	}
	if file.AgentCommands != nil {
		settings.AgentCommands = *file.AgentCommands
	} else {
		// Migrate from legacy fields
		var provider, model, sandbox string
		if file.AgentProvider != nil {
			provider = *file.AgentProvider
		}
		if file.AgentModel != nil {
			model = *file.AgentModel
		}
		if file.AgentSandbox != nil {
			sandbox = *file.AgentSandbox
		}
		settings.AgentCommands = migrateAgentCommands(provider, model, sandbox)
	}
	if file.AgentWorkspace != nil {
		settings.AgentWorkspace = *file.AgentWorkspace
	}

	return settings, nil
}

// migrateAgentCommands builds agent commands from legacy provider/model/sandbox fields.
func migrateAgentCommands(provider, model, sandbox string) []AgentCommand {
	provider = strings.TrimSpace(strings.ToLower(provider))
	model = strings.TrimSpace(model)
	sandbox = strings.TrimSpace(strings.ToLower(sandbox))

	if provider == "" {
		return DefaultAgentCommands()
	}

	switch provider {
	case "claude":
		return migrateClaude(model, sandbox)
	case "cursor":
		return migrateCursor(model, sandbox)
	default:
		return DefaultAgentCommands()
	}
}

// migrateClaude builds agent commands from legacy Claude provider fields.
func migrateClaude(model, sandbox string) []AgentCommand {
	var parts []string
	parts = append(parts, "claude")
	if model != "" {
		parts = append(parts, "--model", model)
	}
	switch sandbox {
	case "dangerously-skip-permissions":
		parts = append(parts, "--dangerously-skip-permissions")
	}
	parts = append(parts, "{prompt}")
	return []AgentCommand{
		{Name: "Claude (migrated)", Command: strings.Join(parts, " ")},
	}
}

// migrateCursor builds agent commands from legacy Cursor provider fields.
func migrateCursor(model, sandbox string) []AgentCommand {
	var parts []string
	parts = append(parts, "cursor-agent")
	if sandbox != "" {
		parts = append(parts, "--sandbox", sandbox)
	}
	if model != "" {
		parts = append(parts, "--model", model)
	}
	parts = append(parts, "{prompt}")
	return []AgentCommand{
		{Name: "Cursor (migrated)", Command: strings.Join(parts, " ")},
	}
}

// SaveSettings writes settings to a JSON file, creating directories as needed.
func SaveSettings(path string, settings Settings) error {
	if path == "" {
		return fmt.Errorf("settings path is empty")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write settings file: %w", err)
	}

	return nil
}

// parseDuration parses a duration string with a labeled error message.
func parseDuration(value string, label string) (time.Duration, error) {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s value %q: %w", label, value, err)
	}

	return duration, nil
}

// validatePageSize validates the allowed page size range.
func validatePageSize(pageSize int, label string) error {
	if pageSize < 1 || pageSize > 250 {
		return fmt.Errorf("%s must be between 1 and 250, got %d", label, pageSize)
	}

	return nil
}

// validateLogLevel validates the allowed log level values.
func validateLogLevel(logLevel string, label string) error {
	switch logLevel {
	case "debug", "info", "warning", "error":
		return nil
	default:
		return fmt.Errorf("invalid %s value %q: must be debug, info, warning, or error", label, logLevel)
	}
}

// validateTheme validates the allowed theme values.
func validateTheme(theme string, label string) error {
	switch theme {
	case ThemeLinear, ThemeHighContrast, ThemeColorBlind:
		return nil
	default:
		return fmt.Errorf("invalid %s value %q: must be linear, high_contrast, or color_blind", label, theme)
	}
}

// validateDensity validates the allowed density values.
func validateDensity(density string, label string) error {
	switch density {
	case DensityComfortable, DensityCompact:
		return nil
	default:
		return fmt.Errorf("invalid %s value %q: must be comfortable or compact", label, density)
	}
}
