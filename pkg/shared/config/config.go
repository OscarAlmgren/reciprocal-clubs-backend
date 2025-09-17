package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Service    ServiceConfig    `mapstructure:"service"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	NATS       NATSConfig       `mapstructure:"nats"`
	Auth       AuthConfig       `mapstructure:"auth"`
	Hanko      HankoConfig      `mapstructure:"hanko"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
	Logging    LoggingConfig    `mapstructure:"logging"`
}

// ServiceConfig holds service-specific configuration
type ServiceConfig struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	Environment string `mapstructure:"environment"`
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	GRPCPort    int    `mapstructure:"grpc_port"`
	Timeout     int    `mapstructure:"timeout"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	Database        string `mapstructure:"database"`
	SSLMode         string `mapstructure:"ssl_mode"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	Database int    `mapstructure:"database"`
}

// NATSConfig holds NATS configuration
type NATSConfig struct {
	URL             string   `mapstructure:"url"`
	ClusterID       string   `mapstructure:"cluster_id"`
	ClientID        string   `mapstructure:"client_id"`
	Servers         []string `mapstructure:"servers"`
	MaxReconnect    int      `mapstructure:"max_reconnect"`
	ReconnectWait   int      `mapstructure:"reconnect_wait"`
	ConnectTimeout  int      `mapstructure:"connect_timeout"`
	RequestTimeout  int      `mapstructure:"request_timeout"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret     string `mapstructure:"jwt_secret"`
	JWTExpiration int    `mapstructure:"jwt_expiration"`
	Issuer        string `mapstructure:"issuer"`
	Audience      string `mapstructure:"audience"`
}

// HankoConfig holds Hanko authentication service configuration
type HankoConfig struct {
	BaseURL    string `mapstructure:"base_url"`
	APIKey     string `mapstructure:"api_key"`
	Timeout    int    `mapstructure:"timeout"`
	RetryCount int    `mapstructure:"retry_count"`
}

// MonitoringConfig holds monitoring configuration
type MonitoringConfig struct {
	MetricsPath    string `mapstructure:"metrics_path"`
	MetricsPort    int    `mapstructure:"metrics_port"`
	HealthPath     string `mapstructure:"health_path"`
	ReadinessPath  string `mapstructure:"readiness_path"`
	EnableMetrics  bool   `mapstructure:"enable_metrics"`
	EnableTracing  bool   `mapstructure:"enable_tracing"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	TimeFormat string `mapstructure:"time_format"`
}

// Load loads configuration from environment variables and config files
func Load(serviceName string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("../config")
	viper.AddConfigPath("../../config")
	viper.AddConfigPath("/etc/" + serviceName)
	
	// Environment variables
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix(strings.ToUpper(serviceName))

	// Set defaults
	setDefaults(serviceName)

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate required fields
	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

func setDefaults(serviceName string) {
	// Service defaults
	viper.SetDefault("service.name", serviceName)
	viper.SetDefault("service.version", "1.0.0")
	viper.SetDefault("service.environment", "development")
	viper.SetDefault("service.host", "0.0.0.0")
	viper.SetDefault("service.port", 8080)
	viper.SetDefault("service.grpc_port", 9090)
	viper.SetDefault("service.timeout", 30)

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.database", serviceName)
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", 300)

	// Redis defaults
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.database", 0)

	// NATS defaults
	viper.SetDefault("nats.url", "nats://localhost:4222")
	viper.SetDefault("nats.cluster_id", "reciprocal-clubs")
	viper.SetDefault("nats.client_id", serviceName)
	viper.SetDefault("nats.max_reconnect", -1)
	viper.SetDefault("nats.reconnect_wait", 2)
	viper.SetDefault("nats.connect_timeout", 5)
	viper.SetDefault("nats.request_timeout", 10)

	// Auth defaults
	viper.SetDefault("auth.jwt_secret", "your-secret-key")
	viper.SetDefault("auth.jwt_expiration", 3600)
	viper.SetDefault("auth.issuer", "reciprocal-clubs")
	viper.SetDefault("auth.audience", "reciprocal-clubs")

	// Monitoring defaults
	viper.SetDefault("monitoring.metrics_path", "/metrics")
	viper.SetDefault("monitoring.metrics_port", 2112)
	viper.SetDefault("monitoring.health_path", "/health")
	viper.SetDefault("monitoring.readiness_path", "/ready")
	viper.SetDefault("monitoring.enable_metrics", true)
	viper.SetDefault("monitoring.enable_tracing", false)

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output", "stdout")
	viper.SetDefault("logging.time_format", "2006-01-02T15:04:05.000Z")
}

func validate(config *Config) error {
	if config.Service.Name == "" {
		return fmt.Errorf("service name is required")
	}
	if config.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if config.Auth.JWTSecret == "" {
		return fmt.Errorf("JWT secret is required")
	}
	return nil
}

// GetDSN returns the database connection string
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

// GetRedisAddr returns the Redis address
func (c *RedisConfig) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}