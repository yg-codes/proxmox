package config

import (
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Config holds application configuration
type Config struct {
	// Proxmox connection settings
	Proxmox ProxmoxConfig `mapstructure:"proxmox"`

	// Operation settings
	Operations OperationsConfig `mapstructure:"operations"`

	// Logging settings
	Logging LoggingConfig `mapstructure:"logging"`

	// CLI settings
	CLI CLIConfig `mapstructure:"cli"`
}

// ProxmoxConfig holds Proxmox API configuration
type ProxmoxConfig struct {
	Host       string        `mapstructure:"host"`
	Port       int           `mapstructure:"port"`
	Username   string        `mapstructure:"username"`
	Password   string        `mapstructure:"password"`
	TokenName  string        `mapstructure:"token_name"`
	TokenValue string        `mapstructure:"token_value"`
	VerifySSL  bool          `mapstructure:"verify_ssl"`
	Timeout    time.Duration `mapstructure:"timeout"`
}

// OperationsConfig holds operation-specific configuration
type OperationsConfig struct {
	MaxConcurrentSnapshots int           `mapstructure:"max_concurrent_snapshots"`
	MaxConcurrentVMOps     int           `mapstructure:"max_concurrent_vm_ops"`
	DefaultVMState         bool          `mapstructure:"default_vm_state"`
	SnapshotNameMaxLength  int           `mapstructure:"snapshot_name_max_length"`
	TaskTimeout            time.Duration `mapstructure:"task_timeout"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// CLIConfig holds CLI-specific configuration
type CLIConfig struct {
	BatchMode    bool `mapstructure:"batch_mode"`
	AutoConfirm  bool `mapstructure:"auto_confirm"`
	ColorOutput  bool `mapstructure:"color_output"`
	ProgressBars bool `mapstructure:"progress_bars"`
}

// Default configuration values
var defaultConfig = Config{
	Proxmox: ProxmoxConfig{
		Port:      8006,
		VerifySSL: false,
		Timeout:   30 * time.Second,
	},
	Operations: OperationsConfig{
		MaxConcurrentSnapshots: 2,
		MaxConcurrentVMOps:     3,
		DefaultVMState:         false,
		SnapshotNameMaxLength:  40,
		TaskTimeout:            300 * time.Second,
	},
	Logging: LoggingConfig{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	},
	CLI: CLIConfig{
		BatchMode:    false,
		AutoConfirm:  false,
		ColorOutput:  true,
		ProgressBars: true,
	},
}

// LoadConfig loads configuration from various sources
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigName("proxmox-snapshot-manager")
	viper.SetConfigType("yaml")

	// Add config search paths
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.config/proxmox-snapshot-manager")
		viper.AddConfigPath("/etc/proxmox-snapshot-manager")
	}

	// Set environment variable prefix
	viper.SetEnvPrefix("PSM")
	viper.AutomaticEnv()

	// Bind environment variables
	viper.BindEnv("proxmox.host", "PVE_HOST")
	viper.BindEnv("proxmox.username", "PVE_USER")
	viper.BindEnv("proxmox.password", "PVE_PASSWORD")
	viper.BindEnv("proxmox.token_name", "PVE_TOKEN_NAME")
	viper.BindEnv("proxmox.token_value", "PVE_TOKEN_VALUE")

	// Set defaults
	setDefaults()

	// Read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found is OK, we'll use defaults + env vars
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values in Viper
func setDefaults() {
	// Proxmox defaults
	viper.SetDefault("proxmox.port", defaultConfig.Proxmox.Port)
	viper.SetDefault("proxmox.verify_ssl", defaultConfig.Proxmox.VerifySSL)
	viper.SetDefault("proxmox.timeout", defaultConfig.Proxmox.Timeout)

	// Operations defaults
	viper.SetDefault("operations.max_concurrent_snapshots", defaultConfig.Operations.MaxConcurrentSnapshots)
	viper.SetDefault("operations.max_concurrent_vm_ops", defaultConfig.Operations.MaxConcurrentVMOps)
	viper.SetDefault("operations.default_vm_state", defaultConfig.Operations.DefaultVMState)
	viper.SetDefault("operations.snapshot_name_max_length", defaultConfig.Operations.SnapshotNameMaxLength)
	viper.SetDefault("operations.task_timeout", defaultConfig.Operations.TaskTimeout)

	// Logging defaults
	viper.SetDefault("logging.level", defaultConfig.Logging.Level)
	viper.SetDefault("logging.format", defaultConfig.Logging.Format)
	viper.SetDefault("logging.output", defaultConfig.Logging.Output)

	// CLI defaults
	viper.SetDefault("cli.batch_mode", defaultConfig.CLI.BatchMode)
	viper.SetDefault("cli.auto_confirm", defaultConfig.CLI.AutoConfirm)
	viper.SetDefault("cli.color_output", defaultConfig.CLI.ColorOutput)
	viper.SetDefault("cli.progress_bars", defaultConfig.CLI.ProgressBars)
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate Proxmox configuration
	if c.Proxmox.Host == "" {
		return fmt.Errorf("proxmox host is required")
	}

	if c.Proxmox.Username == "" {
		return fmt.Errorf("proxmox username is required")
	}

	// Check authentication method
	hasToken := c.Proxmox.TokenName != "" && c.Proxmox.TokenValue != ""
	hasPassword := c.Proxmox.Password != ""

	if !hasToken && !hasPassword {
		return fmt.Errorf("either token authentication or password is required")
	}

	if c.Proxmox.Port <= 0 || c.Proxmox.Port > 65535 {
		return fmt.Errorf("proxmox port must be between 1 and 65535")
	}

	// Validate operations configuration
	if c.Operations.MaxConcurrentSnapshots <= 0 {
		return fmt.Errorf("max_concurrent_snapshots must be greater than 0")
	}

	if c.Operations.MaxConcurrentVMOps <= 0 {
		return fmt.Errorf("max_concurrent_vm_ops must be greater than 0")
	}

	if c.Operations.SnapshotNameMaxLength <= 0 {
		return fmt.Errorf("snapshot_name_max_length must be greater than 0")
	}

	// Validate logging configuration
	validLevels := map[string]bool{
		"panic": true, "fatal": true, "error": true,
		"warn": true, "info": true, "debug": true, "trace": true,
	}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", c.Logging.Level)
	}

	validFormats := map[string]bool{"text": true, "json": true}
	if !validFormats[c.Logging.Format] {
		return fmt.Errorf("invalid log format: %s", c.Logging.Format)
	}

	return nil
}

// SetupLogger configures and returns a logger based on the config
func (c *Config) SetupLogger() *logrus.Logger {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(c.Logging.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set log format
	if c.Logging.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			ForceColors:     c.CLI.ColorOutput,
		})
	}

	// Set output
	switch c.Logging.Output {
	case "stderr":
		logger.SetOutput(os.Stderr)
	case "stdout":
		logger.SetOutput(os.Stdout)
	default:
		// Try to open as file
		if file, err := os.OpenFile(c.Logging.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
			logger.SetOutput(file)
		} else {
			logger.SetOutput(os.Stdout)
			logger.Warnf("Could not open log file %s, using stdout: %v", c.Logging.Output, err)
		}
	}

	return logger
}

// IsBatchMode returns true if running in batch mode
func (c *Config) IsBatchMode() bool {
	return c.CLI.BatchMode
}

// IsAutoConfirm returns true if auto-confirm is enabled
func (c *Config) IsAutoConfirm() bool {
	return c.CLI.AutoConfirm
}

// GetMaxConcurrentOperations returns max concurrent operations based on operation type
func (c *Config) GetMaxConcurrentOperations(operationType string) int {
	switch operationType {
	case "snapshot":
		return c.Operations.MaxConcurrentSnapshots
	case "vm":
		return c.Operations.MaxConcurrentVMOps
	default:
		return c.Operations.MaxConcurrentSnapshots
	}
}

// CreateSampleConfig creates a sample configuration file
func CreateSampleConfig(path string) error {
	sampleConfig := `# Proxmox Snapshot Manager Configuration
# Copy to ~/.config/proxmox-snapshot-manager/proxmox-snapshot-manager.yaml

proxmox:
  host: "your-proxmox-host.example.com"
  port: 8006
  username: "user@pam"
  # Use either token authentication (recommended):
  token_name: "api-token-name"
  token_value: "your-token-value"
  # Or password authentication:
  # password: "your-password"
  verify_ssl: false
  timeout: 30s

operations:
  max_concurrent_snapshots: 2
  max_concurrent_vm_ops: 3
  default_vm_state: false
  snapshot_name_max_length: 40
  task_timeout: 300s

logging:
  level: "info"  # panic, fatal, error, warn, info, debug, trace
  format: "text" # text or json
  output: "stdout" # stdout, stderr, or file path

cli:
  batch_mode: false
  auto_confirm: false
  color_output: true
  progress_bars: true
`

	return os.WriteFile(path, []byte(sampleConfig), 0644)
}
