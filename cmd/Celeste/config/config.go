// Package config provides configuration management for Celeste CLI.
package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/whykusanagi/celesteCLI/cmd/Celeste/skills"
)

// Config holds all configuration for Celeste CLI.
type Config struct {
	// API settings
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
	Timeout int    `json:"timeout"` // seconds

	// Persona settings
	SkipPersonaPrompt bool `json:"skip_persona_prompt"`

	// Streaming settings
	SimulateTyping bool `json:"simulate_typing"`
	TypingSpeed    int  `json:"typing_speed"` // chars per second

	// Venice.ai settings (for NSFW mode)
	VeniceAPIKey  string `json:"venice_api_key,omitempty"`
	VeniceBaseURL string `json:"venice_base_url,omitempty"`
	VeniceModel   string `json:"venice_model,omitempty"`

	// Tarot settings
	TarotFunctionURL string `json:"tarot_function_url,omitempty"`
	TarotAuthToken   string `json:"tarot_auth_token,omitempty"`

	// Twitter settings
	TwitterBearerToken      string `json:"twitter_bearer_token,omitempty"`
	TwitterAPIKey           string `json:"twitter_api_key,omitempty"`
	TwitterAPISecret        string `json:"twitter_api_secret,omitempty"`
	TwitterAccessToken      string `json:"twitter_access_token,omitempty"`
	TwitterAccessTokenSecret string `json:"twitter_access_token_secret,omitempty"`
}

// DefaultConfig returns a config with default values.
func DefaultConfig() *Config {
	return &Config{
		BaseURL:           "https://api.openai.com/v1",
		Model:             "gpt-4o-mini",
		Timeout:           60,
		SkipPersonaPrompt: false,
		SimulateTyping:    true,
		TypingSpeed:       40,
		VeniceBaseURL:     "https://api.venice.ai/api/v1",
		VeniceModel:       "venice-uncensored",
	}
}

// Paths returns the configuration directory and file paths.
func Paths() (configDir, configFile, secretsFile, oldConfigFile string) {
	homeDir, _ := os.UserHomeDir()
	configDir = filepath.Join(homeDir, ".celeste")
	configFile = filepath.Join(configDir, "config.json")
	secretsFile = filepath.Join(configDir, "secrets.json")
	oldConfigFile = filepath.Join(homeDir, ".celesteAI")
	return
}

// NamedConfigPath returns the path for a named config file.
// If name is empty, returns the default config path.
func NamedConfigPath(name string) string {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".celeste")
	if name == "" {
		return filepath.Join(configDir, "config.json")
	}
	return filepath.Join(configDir, fmt.Sprintf("config.%s.json", name))
}

// LoadNamed loads configuration from a named config file.
// If name is empty, loads the default config.
func LoadNamed(name string) (*Config, error) {
	if name == "" {
		return Load()
	}

	config := DefaultConfig()
	configPath := NamedConfigPath(name)
	configDir, _, secretsFile, _ := Paths()

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Load named config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("config '%s' not found at %s: %w", name, configPath, err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config '%s': %w", name, err)
	}

	// Load shared secrets file (for common secrets like Venice, Twitter)
	if data, err := os.ReadFile(secretsFile); err == nil {
		var secrets Config
		if err := json.Unmarshal(data, &secrets); err == nil {
			// Only merge secrets that aren't already set in the named config
			if config.VeniceAPIKey == "" && secrets.VeniceAPIKey != "" {
				config.VeniceAPIKey = secrets.VeniceAPIKey
			}
			if config.TarotAuthToken == "" && secrets.TarotAuthToken != "" {
				config.TarotAuthToken = secrets.TarotAuthToken
			}
			if config.TwitterBearerToken == "" && secrets.TwitterBearerToken != "" {
				config.TwitterBearerToken = secrets.TwitterBearerToken
			}
		}
	}

	// Override with environment variables
	if apiKey := os.Getenv("CELESTE_API_KEY"); apiKey != "" {
		config.APIKey = apiKey
	}
	if endpoint := os.Getenv("CELESTE_API_ENDPOINT"); endpoint != "" {
		config.BaseURL = endpoint
	}

	return config, nil
}

// ListConfigs returns all available config names.
func ListConfigs() ([]string, error) {
	configDir, _, _, _ := Paths()
	
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return nil, err
	}

	var configs []string
	for _, entry := range entries {
		name := entry.Name()
		if name == "config.json" {
			configs = append(configs, "default")
		} else if len(name) > 12 && name[:7] == "config." && name[len(name)-5:] == ".json" {
			// Extract name from config.<name>.json
			configName := name[7 : len(name)-5]
			configs = append(configs, configName)
		}
	}

	return configs, nil
}

// Load loads configuration from file and environment.
func Load() (*Config, error) {
	config := DefaultConfig()
	configDir, configFile, secretsFile, oldConfigFile := Paths()

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Try to migrate from old config format
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if _, err := os.Stat(oldConfigFile); err == nil {
			if err := migrateOldConfig(oldConfigFile, configFile, secretsFile); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to migrate old config: %v\n", err)
			}
		}
	}

	// Load main config file
	if data, err := os.ReadFile(configFile); err == nil {
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse config: %w", err)
		}
	}

	// Load secrets file (overwrites sensitive fields)
	if data, err := os.ReadFile(secretsFile); err == nil {
		var secrets Config
		if err := json.Unmarshal(data, &secrets); err == nil {
			if secrets.APIKey != "" {
				config.APIKey = secrets.APIKey
			}
			if secrets.VeniceAPIKey != "" {
				config.VeniceAPIKey = secrets.VeniceAPIKey
			}
			if secrets.TarotAuthToken != "" {
				config.TarotAuthToken = secrets.TarotAuthToken
			}
			if secrets.TwitterBearerToken != "" {
				config.TwitterBearerToken = secrets.TwitterBearerToken
			}
			if secrets.TwitterAPIKey != "" {
				config.TwitterAPIKey = secrets.TwitterAPIKey
			}
			if secrets.TwitterAPISecret != "" {
				config.TwitterAPISecret = secrets.TwitterAPISecret
			}
			if secrets.TwitterAccessToken != "" {
				config.TwitterAccessToken = secrets.TwitterAccessToken
			}
			if secrets.TwitterAccessTokenSecret != "" {
				config.TwitterAccessTokenSecret = secrets.TwitterAccessTokenSecret
			}
		}
	}

	// Override with environment variables
	if apiKey := os.Getenv("CELESTE_API_KEY"); apiKey != "" {
		config.APIKey = apiKey
	}
	if endpoint := os.Getenv("CELESTE_API_ENDPOINT"); endpoint != "" {
		config.BaseURL = endpoint
	}
	if veniceKey := os.Getenv("VENICE_API_KEY"); veniceKey != "" {
		config.VeniceAPIKey = veniceKey
	}
	if tarotToken := os.Getenv("TAROT_AUTH_TOKEN"); tarotToken != "" {
		config.TarotAuthToken = tarotToken
	}

	return config, nil
}

// Save saves configuration to file.
func Save(config *Config) error {
	_, configFile, _, _ := Paths()

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configFile, data, 0644)
}

// SaveSecrets saves sensitive configuration to secrets file.
func SaveSecrets(config *Config) error {
	_, _, secretsFile, _ := Paths()

	secrets := &Config{
		APIKey:                   config.APIKey,
		VeniceAPIKey:             config.VeniceAPIKey,
		TarotAuthToken:           config.TarotAuthToken,
		TwitterBearerToken:       config.TwitterBearerToken,
		TwitterAPIKey:            config.TwitterAPIKey,
		TwitterAPISecret:         config.TwitterAPISecret,
		TwitterAccessToken:       config.TwitterAccessToken,
		TwitterAccessTokenSecret: config.TwitterAccessTokenSecret,
	}

	data, err := json.MarshalIndent(secrets, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal secrets: %w", err)
	}

	return os.WriteFile(secretsFile, data, 0600) // More restrictive permissions for secrets
}

// migrateOldConfig migrates from old .celesteAI format to new JSON format.
func migrateOldConfig(oldPath, configPath, secretsPath string) error {
	data, err := os.ReadFile(oldPath)
	if err != nil {
		return err
	}

	config := DefaultConfig()
	secrets := &Config{}

	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		// Skip comments
		if bytes.HasPrefix(bytes.TrimSpace(line), []byte("#")) {
			continue
		}
		parts := bytes.SplitN(line, []byte("="), 2)
		if len(parts) != 2 {
			continue
		}
		key := string(bytes.TrimSpace(parts[0]))
		val := string(bytes.TrimSpace(parts[1]))

		switch key {
		case "endpoint":
			config.BaseURL = val
		case "api_key":
			secrets.APIKey = val
		case "model":
			config.Model = val
		case "venice_api_key":
			secrets.VeniceAPIKey = val
		case "venice_base_url":
			config.VeniceBaseURL = val
		case "venice_model":
			config.VeniceModel = val
		case "tarot_function_url":
			config.TarotFunctionURL = val
		case "tarot_auth_token":
			secrets.TarotAuthToken = val
		case "twitter_bearer_token":
			secrets.TwitterBearerToken = val
		case "twitter_api_key":
			secrets.TwitterAPIKey = val
		case "twitter_api_secret":
			secrets.TwitterAPISecret = val
		case "twitter_access_token":
			secrets.TwitterAccessToken = val
		case "twitter_access_token_secret":
			secrets.TwitterAccessTokenSecret = val
		}
	}

	// Save new config
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configData, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return err
	}

	secretsData, _ := json.MarshalIndent(secrets, "", "  ")
	if err := os.WriteFile(secretsPath, secretsData, 0600); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Migrated configuration from %s to %s\n", oldPath, configPath)
	return nil
}

// ConfigLoader implements skills.ConfigLoader interface.
type ConfigLoader struct {
	config *Config
}

// NewConfigLoader creates a new config loader.
func NewConfigLoader(config *Config) *ConfigLoader {
	return &ConfigLoader{config: config}
}

// GetTarotConfig returns tarot configuration.
func (l *ConfigLoader) GetTarotConfig() (skills.TarotConfig, error) {
	if l.config.TarotAuthToken == "" {
		return skills.TarotConfig{}, fmt.Errorf("tarot auth token not configured")
	}

	url := l.config.TarotFunctionURL
	if url == "" {
		url = "https://faas-nyc1-2ef2e6cc.doserverless.co/api/v1/namespaces/fn-30b193db-d334-4dab-b5cd-ab49067f88cc/actions/tarot/logic?blocking=true&result=true"
	}

	return skills.TarotConfig{
		FunctionURL: url,
		AuthToken:   l.config.TarotAuthToken,
	}, nil
}

// GetVeniceConfig returns Venice.ai configuration.
func (l *ConfigLoader) GetVeniceConfig() (skills.VeniceConfig, error) {
	if l.config.VeniceAPIKey == "" {
		return skills.VeniceConfig{}, fmt.Errorf("Venice.ai API key not configured")
	}

	baseURL := l.config.VeniceBaseURL
	if baseURL == "" {
		baseURL = "https://api.venice.ai/api/v1"
	}

	model := l.config.VeniceModel
	if model == "" {
		model = "venice-uncensored"
	}

	return skills.VeniceConfig{
		APIKey:   l.config.VeniceAPIKey,
		BaseURL:  baseURL,
		Model:    model,
		Upscaler: "upscaler",
	}, nil
}

// GetTimeout returns the configured timeout as a duration.
func (c *Config) GetTimeout() time.Duration {
	if c.Timeout <= 0 {
		return 60 * time.Second
	}
	return time.Duration(c.Timeout) * time.Second
}

