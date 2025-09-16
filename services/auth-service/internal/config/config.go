package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the auth service
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Hanko    HankoConfig    `mapstructure:"hanko"`
	NATS     NATSConfig     `mapstructure:"nats"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Environment    string        `mapstructure:"environment"`
	HTTPPort       int           `mapstructure:"http_port"`
	GRPCPort       int           `mapstructure:"grpc_port"`
	ReadTimeout    time.Duration `mapstructure:"read_timeout"`
	WriteTimeout   time.Duration `mapstructure:"write_timeout"`
	IdleTimeout    time.Duration `mapstructure:"idle_timeout"`
	ShutdownGrace  time.Duration `mapstructure:"shutdown_grace"`
	EnableCORS     bool          `mapstructure:"enable_cors"`
	TrustedProxies []string      `mapstructure:"trusted_proxies"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
	AutoMigrate     bool          `mapstructure:"auto_migrate"`
}

// HankoConfig holds Hanko integration configuration
type HankoConfig struct {
	Enabled     bool          `mapstructure:"enabled"`
	URL         string        `mapstructure:"url"`
	APIKey      string        `mapstructure:"api_key"`
	ProjectID   string        `mapstructure:"project_id"`
	Timeout     time.Duration `mapstructure:"timeout"`
	MaxRetries  int           `mapstructure:"max_retries"`
	WebhookURL  string        `mapstructure:"webhook_url"`
	UseMock     bool          `mapstructure:"use_mock"`
}

// NATSConfig holds NATS configuration
type NATSConfig struct {
	URL           string        `mapstructure:"url"`
	ClusterID     string        `mapstructure:"cluster_id"`
	ClientID      string        `mapstructure:"client_id"`
	Timeout       time.Duration `mapstructure:"timeout"`
	MaxReconnect  int           `mapstructure:"max_reconnect"`
	ReconnectWait time.Duration `mapstructure:"reconnect_wait"`
	PingInterval  time.Duration `mapstructure:"ping_interval"`
	MaxPingsOut   int           `mapstructure:"max_pings_out"`
	Subject       SubjectConfig `mapstructure:"subject"`
}

// SubjectConfig holds NATS subject configuration
type SubjectConfig struct {
	UserEvents string `mapstructure:"user_events"`
	AuthEvents string `mapstructure:"auth_events"`
	AuditLogs  string `mapstructure:"audit_logs"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	SessionDuration       time.Duration `mapstructure:"session_duration"`
	TokenSecretKey        string        `mapstructure:"token_secret_key"`
	PasswordMinLength     int           `mapstructure:"password_min_length"`
	MaxLoginAttempts      int           `mapstructure:"max_login_attempts"`
	LoginAttemptWindow    time.Duration `mapstructure:"login_attempt_window"`
	AccountLockoutDuration time.Duration `mapstructure:"account_lockout_duration"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level        string `mapstructure:"level"`
	Format       string `mapstructure:"format"` // json, text
	Output       string `mapstructure:"output"` // stdout, stderr, file
	File         string `mapstructure:"file"`
	MaxSize      int    `mapstructure:"max_size"`      // MB
	MaxBackups   int    `mapstructure:"max_backups"`
	MaxAge       int    `mapstructure:"max_age"`       // days
	Compress     bool   `mapstructure:"compress"`
	EnableCaller bool   `mapstructure:"enable_caller"`
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Port      int    `mapstructure:"port"`
	Path      string `mapstructure:"path"`
	Namespace string `mapstructure:"namespace"`
}

// Load loads configuration from various sources
func Load() (*Config, error) {
	// Set configuration file name and paths
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/auth-service")

	// Set environment variable prefix
	viper.SetEnvPrefix("AUTH_SERVICE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set defaults
	setDefaults()

	// Read configuration file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, continue with defaults and env vars
	}

	// Unmarshal configuration
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate configuration
	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.environment", "development")
	viper.SetDefault("server.http_port", 8080)
	viper.SetDefault("server.grpc_port", 8081)
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.idle_timeout", "120s")
	viper.SetDefault("server.shutdown_grace", "30s")
	viper.SetDefault("server.enable_cors", true)
	viper.SetDefault("server.trusted_proxies", []string{})

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.username", "postgres")
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.database", "auth_service")
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 25)
	viper.SetDefault("database.conn_max_lifetime", "5m")
	viper.SetDefault("database.conn_max_idle_time", "5m")
	viper.SetDefault("database.auto_migrate", true)

	// Hanko defaults
	viper.SetDefault("hanko.enabled", true)
	viper.SetDefault("hanko.url", "http://localhost:8000")
	viper.SetDefault("hanko.timeout", "30s")
	viper.SetDefault("hanko.max_retries", 3)
	viper.SetDefault("hanko.use_mock", false)

	// NATS defaults
	viper.SetDefault("nats.url", "nats://localhost:4222")
	viper.SetDefault("nats.cluster_id", "reciprocal-clubs")
	viper.SetDefault("nats.client_id", "auth-service")
	viper.SetDefault("nats.timeout", "30s")
	viper.SetDefault("nats.max_reconnect", 10)
	viper.SetDefault("nats.reconnect_wait", "2s")
	viper.SetDefault("nats.ping_interval", "20s")
	viper.SetDefault("nats.max_pings_out", 2)
	viper.SetDefault("nats.subject.user_events", "users.events")
	viper.SetDefault("nats.subject.auth_events", "auth.events")
	viper.SetDefault("nats.subject.audit_logs", "audit.logs")

	// Auth defaults
	viper.SetDefault("auth.session_duration", "24h")
	viper.SetDefault("auth.password_min_length", 8)
	viper.SetDefault("auth.max_login_attempts", 5)
	viper.SetDefault("auth.login_attempt_window", "15m")
	viper.SetDefault("auth.account_lockout_duration", "30m")

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output", "stdout")
	viper.SetDefault("logging.max_size", 100)
	viper.SetDefault("logging.max_backups", 3)
	viper.SetDefault("logging.max_age", 28)
	viper.SetDefault("logging.compress", true)
	viper.SetDefault("logging.enable_caller", true)

	// Metrics defaults
	viper.SetDefault("metrics.enabled", true)
	viper.SetDefault("metrics.port", 9090)
	viper.SetDefault("metrics.path", "/metrics")
	viper.SetDefault("metrics.namespace", "auth_service")
}

// validate validates the configuration
func validate(config *Config) error {
	// Validate server configuration
	if config.Server.HTTPPort <= 0 || config.Server.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d", config.Server.HTTPPort)
	}

	if config.Server.GRPCPort <= 0 || config.Server.GRPCPort > 65535 {
		return fmt.Errorf("invalid gRPC port: %d", config.Server.GRPCPort)
	}

	if config.Server.HTTPPort == config.Server.GRPCPort {
		return fmt.Errorf("HTTP and gRPC ports cannot be the same: %d", config.Server.HTTPPort)
	}

	// Validate database configuration
	if config.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if config.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}

	if config.Database.MaxOpenConns <= 0 {
		return fmt.Errorf("invalid max open connections: %d", config.Database.MaxOpenConns)
	}

	if config.Database.MaxIdleConns < 0 {
		return fmt.Errorf("invalid max idle connections: %d", config.Database.MaxIdleConns)
	}

	// Validate Hanko configuration
	if config.Hanko.Enabled && config.Hanko.URL == "" {
		return fmt.Errorf("Hanko URL is required when Hanko is enabled")
	}

	// Validate NATS configuration
	if config.NATS.URL == "" {
		return fmt.Errorf("NATS URL is required")
	}

	// Validate auth configuration
	if config.Auth.TokenSecretKey == "" {
		return fmt.Errorf("token secret key is required")
	}

	if len(config.Auth.TokenSecretKey) < 32 {
		return fmt.Errorf("token secret key must be at least 32 characters long")
	}

	if config.Auth.PasswordMinLength < 8 {
		return fmt.Errorf("password minimum length must be at least 8")
	}

	// Validate logging configuration
	validLogLevels := []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}
	if !contains(validLogLevels, config.Logging.Level) {
		return fmt.Errorf("invalid log level: %s", config.Logging.Level)
	}

	validLogFormats := []string{"json", "text"}
	if !contains(validLogFormats, config.Logging.Format) {
		return fmt.Errorf("invalid log format: %s", config.Logging.Format)
	}

	validLogOutputs := []string{"stdout", "stderr", "file"}
	if !contains(validLogOutputs, config.Logging.Output) {
		return fmt.Errorf("invalid log output: %s", config.Logging.Output)
	}

	if config.Logging.Output == "file" && config.Logging.File == "" {
		return fmt.Errorf("log file path is required when output is file")
	}

	// Validate metrics configuration
	if config.Metrics.Enabled && (config.Metrics.Port <= 0 || config.Metrics.Port > 65535) {
		return fmt.Errorf("invalid metrics port: %d", config.Metrics.Port)
	}

	return nil
}

// GetDatabaseDSN returns the database connection string
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.Username,
		c.Database.Password,
		c.Database.Database,
		c.Database.SSLMode,
	)
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return strings.ToLower(c.Server.Environment) == "production"
}

// IsDevelopment returns true if running in development environment
func (c *Config) IsDevelopment() bool {
	return strings.ToLower(c.Server.Environment) == "development"
}

// IsTest returns true if running in test environment
func (c *Config) IsTest() bool {
	return strings.ToLower(c.Server.Environment) == "test"
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}