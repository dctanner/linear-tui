package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Environment variable names for configuration.
const (
	LinearAPIKeyEnv   = "LINEAR_API_KEY"
	LinearAPIEndpoint = "LINEAR_API_ENDPOINT"
	TimeoutEnv        = "LINEAR_TIMEOUT"
	PageSizeEnv       = "LINEAR_PAGE_SIZE"
	CacheTTLEnv       = "LINEAR_CACHE_TTL"
	LogFileEnv        = "LINEAR_LOG_FILE"
	LogLevelEnv       = "LINEAR_LOG_LEVEL"
)

// Default configuration values.
const (
	DefaultTimeout     = 30 * time.Second
	DefaultPageSize    = 50
	DefaultCacheTTL    = 5 * time.Minute
	DefaultAPIEndpoint = "https://api.linear.app/graphql"
	DefaultLogLevel    = "warning" // debug, info, warning, error
	ThemeLinear        = "linear"
	ThemeHighContrast  = "high_contrast"
	ThemeColorBlind    = "color_blind"
	DefaultTheme       = ThemeLinear
	DensityComfortable = "comfortable"
	DensityCompact     = "compact"
	DefaultDensity     = DensityComfortable
)

// AgentCommand defines a user-configurable agent command.
type AgentCommand struct {
	Name    string `json:"name"`    // Display name, e.g. "Claude (skip permissions)"
	Command string `json:"command"` // Command template with {prompt} placeholder
}

// DefaultAgentCommands returns the default set of agent commands.
func DefaultAgentCommands() []AgentCommand {
	return []AgentCommand{
		{Name: "Claude", Command: "claude {prompt}"},
		{Name: "Claude (skip permissions)", Command: "claude --dangerously-skip-permissions {prompt}"},
	}
}

// getDefaultLogFile returns the default log file path: $HOME/.linear-tui/app.log
func getDefaultLogFile() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to empty string if home directory cannot be determined
		return ""
	}
	return filepath.Join(homeDir, ".linear-tui", "app.log")
}

// Config holds runtime configuration for the application.
type Config struct {
	// LinearAPIKey is the API key for authenticating with Linear.
	LinearAPIKey string

	// APIEndpoint is the Linear GraphQL API endpoint (useful for testing).
	APIEndpoint string

	// Timeout is the HTTP request timeout for API calls.
	Timeout time.Duration

	// PageSize is the default number of items to fetch per page.
	PageSize int

	// CacheTTL is the time-to-live for cached team metadata.
	CacheTTL time.Duration

	// LogFile is the path to the log file (empty to disable logging).
	LogFile string

	// LogLevel is the minimum log level (debug, info, warning, error).
	LogLevel string

	// Theme controls the active UI theme.
	Theme string

	// Density controls the UI spacing density.
	Density string

	// AgentCommands is the list of user-configurable agent commands.
	AgentCommands []AgentCommand

	// AgentWorkspace is the default workspace path for agent runs.
	AgentWorkspace string
}

// LoadFromEnv loads configuration from environment variables.
// Returns an error if LINEAR_API_KEY is not set.
// Other values use sensible defaults if not specified.
func LoadFromEnv() (Config, error) {
	apiKey := os.Getenv(LinearAPIKeyEnv)
	if apiKey == "" {
		return Config{}, fmt.Errorf("%s environment variable is not set", LinearAPIKeyEnv)
	}

	cfg := Config{
		LinearAPIKey:   apiKey,
		APIEndpoint:    DefaultAPIEndpoint,
		Timeout:        DefaultTimeout,
		PageSize:       DefaultPageSize,
		CacheTTL:       DefaultCacheTTL,
		LogFile:        getDefaultLogFile(), // Default: $HOME/.linear-tui/app.log
		LogLevel:       DefaultLogLevel,
		Theme:          DefaultTheme,
		Density:        DefaultDensity,
		AgentCommands:  DefaultAgentCommands(),
		AgentWorkspace: "",
	}

	// Parse optional API endpoint override.
	if endpoint := os.Getenv(LinearAPIEndpoint); endpoint != "" {
		cfg.APIEndpoint = endpoint
	}

	// Parse optional timeout.
	if timeoutStr := os.Getenv(TimeoutEnv); timeoutStr != "" {
		timeout, err := parseDuration(timeoutStr, TimeoutEnv)
		if err != nil {
			return Config{}, err
		}
		cfg.Timeout = timeout
	}

	// Parse optional page size.
	if pageSizeStr := os.Getenv(PageSizeEnv); pageSizeStr != "" {
		pageSize, err := strconv.Atoi(pageSizeStr)
		if err != nil {
			return Config{}, fmt.Errorf("invalid %s value %q: %w", PageSizeEnv, pageSizeStr, err)
		}
		if err := validatePageSize(pageSize, PageSizeEnv); err != nil {
			return Config{}, err
		}
		cfg.PageSize = pageSize
	}

	// Parse optional cache TTL.
	if cacheTTLStr := os.Getenv(CacheTTLEnv); cacheTTLStr != "" {
		cacheTTL, err := parseDuration(cacheTTLStr, CacheTTLEnv)
		if err != nil {
			return Config{}, err
		}
		cfg.CacheTTL = cacheTTL
	}

	// Parse optional log file path.
	// If LINEAR_LOG_FILE is set to empty string, disable logging.
	// If not set, use default: $HOME/.linear-tui/app.log
	if logFile, ok := os.LookupEnv(LogFileEnv); ok {
		if logFile == "" {
			cfg.LogFile = "" // Explicitly disable logging
		} else {
			cfg.LogFile = logFile
		}
	}
	// If LINEAR_LOG_FILE is not set, cfg.LogFile already has the default value

	// Parse optional log level.
	if logLevel := os.Getenv(LogLevelEnv); logLevel != "" {
		if err := validateLogLevel(logLevel, LogLevelEnv); err != nil {
			return Config{}, err
		}
		cfg.LogLevel = logLevel
	}

	return cfg, nil
}
