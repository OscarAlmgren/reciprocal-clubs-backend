package logging

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"reciprocal-clubs-backend/pkg/shared/config"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger interface defines the logging contract
type Logger interface {
	Debug(msg string, fields map[string]interface{})
	Info(msg string, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Error(msg string, fields map[string]interface{})
	Fatal(msg string, fields map[string]interface{})
	With(fields map[string]interface{}) Logger
	WithContext(ctx context.Context) Logger
}

// ZerologLogger implements Logger interface using zerolog
type ZerologLogger struct {
	logger zerolog.Logger
}

// ContextKey type for context keys
type ContextKey string

const (
	// CorrelationIDKey is the context key for correlation ID
	CorrelationIDKey ContextKey = "correlation_id"
	// UserIDKey is the context key for user ID
	UserIDKey ContextKey = "user_id"
	// ClubIDKey is the context key for club ID
	ClubIDKey ContextKey = "club_id"
	// ServiceKey is the context key for service name
	ServiceKey ContextKey = "service"
)

// NewLogger creates a new logger instance
func NewLogger(cfg *config.LoggingConfig, serviceName string) Logger {
	var output io.Writer = os.Stdout

	// Configure output
	switch strings.ToLower(cfg.Output) {
	case "stdout":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	default:
		if file, err := os.OpenFile(cfg.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
			output = file
		}
	}

	// Configure time format
	zerolog.TimeFieldFormat = cfg.TimeFormat
	if cfg.TimeFormat == "" {
		zerolog.TimeFieldFormat = time.RFC3339Nano
	}

	var logger zerolog.Logger

	// Configure format
	switch strings.ToLower(cfg.Format) {
	case "pretty", "console":
		logger = zerolog.New(zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: cfg.TimeFormat,
		})
	default:
		logger = zerolog.New(output)
	}

	// Set level
	level, err := zerolog.ParseLevel(strings.ToLower(cfg.Level))
	if err != nil {
		level = zerolog.InfoLevel
	}
	logger = logger.Level(level)

	// Add timestamp and service name
	logger = logger.With().
		Timestamp().
		Str("service", serviceName).
		Logger()

	// Set global logger
	log.Logger = logger

	return &ZerologLogger{logger: logger}
}

// Debug logs a debug message
func (l *ZerologLogger) Debug(msg string, fields map[string]interface{}) {
	event := l.logger.Debug()
	l.addFields(event, fields)
	event.Msg(msg)
}

// Info logs an info message
func (l *ZerologLogger) Info(msg string, fields map[string]interface{}) {
	event := l.logger.Info()
	l.addFields(event, fields)
	event.Msg(msg)
}

// Warn logs a warning message
func (l *ZerologLogger) Warn(msg string, fields map[string]interface{}) {
	event := l.logger.Warn()
	l.addFields(event, fields)
	event.Msg(msg)
}

// Error logs an error message
func (l *ZerologLogger) Error(msg string, fields map[string]interface{}) {
	event := l.logger.Error()
	l.addFields(event, fields)
	event.Msg(msg)
}

// Fatal logs a fatal message and exits
func (l *ZerologLogger) Fatal(msg string, fields map[string]interface{}) {
	event := l.logger.Fatal()
	l.addFields(event, fields)
	event.Msg(msg)
}

// With returns a new logger with additional fields
func (l *ZerologLogger) With(fields map[string]interface{}) Logger {
	logger := l.logger.With()
	for key, value := range fields {
		logger = logger.Interface(key, value)
	}
	return &ZerologLogger{logger: logger.Logger()}
}

// WithContext returns a new logger with context fields
func (l *ZerologLogger) WithContext(ctx context.Context) Logger {
	logger := l.logger.With()

	// Add correlation ID if present
	if correlationID := ctx.Value(CorrelationIDKey); correlationID != nil {
		logger = logger.Str("correlation_id", correlationID.(string))
	}

	// Add user ID if present
	if userID := ctx.Value(UserIDKey); userID != nil {
		logger = logger.Interface("user_id", userID)
	}

	// Add club ID if present
	if clubID := ctx.Value(ClubIDKey); clubID != nil {
		logger = logger.Interface("club_id", clubID)
	}

	return &ZerologLogger{logger: logger.Logger()}
}

// addFields adds fields to the zerolog event
func (l *ZerologLogger) addFields(event *zerolog.Event, fields map[string]interface{}) {
	for key, value := range fields {
		event.Interface(key, value)
	}
}

// ContextWithCorrelationID adds a correlation ID to the context
func ContextWithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, CorrelationIDKey, correlationID)
}

// ContextWithUserID adds a user ID to the context
func ContextWithUserID(ctx context.Context, userID interface{}) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// ContextWithClubID adds a club ID to the context
func ContextWithClubID(ctx context.Context, clubID interface{}) context.Context {
	return context.WithValue(ctx, ClubIDKey, clubID)
}

// ContextWithService adds a service name to the context
func ContextWithService(ctx context.Context, service string) context.Context {
	return context.WithValue(ctx, ServiceKey, service)
}

// GetCorrelationID retrieves the correlation ID from context
func GetCorrelationID(ctx context.Context) string {
	if correlationID := ctx.Value(CorrelationIDKey); correlationID != nil {
		return correlationID.(string)
	}
	return ""
}

// GetUserID retrieves the user ID from context
func GetUserID(ctx context.Context) interface{} {
	return ctx.Value(UserIDKey)
}

// GetClubID retrieves the club ID from context
func GetClubID(ctx context.Context) interface{} {
	return ctx.Value(ClubIDKey)
}

// GetService retrieves the service name from context
func GetService(ctx context.Context) string {
	if service := ctx.Value(ServiceKey); service != nil {
		return service.(string)
	}
	return ""
}

// HTTPRequestLogger is a middleware-friendly logger for HTTP requests
type HTTPRequestLogger struct {
	Logger
}

// NewHTTPRequestLogger creates a new HTTP request logger
func NewHTTPRequestLogger(logger Logger) *HTTPRequestLogger {
	return &HTTPRequestLogger{Logger: logger}
}

// LogRequest logs HTTP request details
func (l *HTTPRequestLogger) LogRequest(ctx context.Context, method, path, userAgent, remoteAddr string, statusCode int, responseTime time.Duration) {
	fields := map[string]interface{}{
		"method":        method,
		"path":          path,
		"status_code":   statusCode,
		"response_time": responseTime.Milliseconds(),
		"user_agent":    userAgent,
		"remote_addr":   remoteAddr,
	}

	contextLogger := l.Logger.WithContext(ctx)

	switch {
	case statusCode >= 500:
		contextLogger.Error("HTTP request completed with server error", fields)
	case statusCode >= 400:
		contextLogger.Warn("HTTP request completed with client error", fields)
	default:
		contextLogger.Info("HTTP request completed", fields)
	}
}

// LogError logs HTTP errors with additional context
func (l *HTTPRequestLogger) LogError(ctx context.Context, err error, method, path string) {
	fields := map[string]interface{}{
		"error":  err.Error(),
		"method": method,
		"path":   path,
	}

	l.Logger.WithContext(ctx).Error("HTTP request error", fields)
}

// GRPCLogger is a logger for gRPC services
type GRPCLogger struct {
	Logger
}

// NewGRPCLogger creates a new gRPC logger
func NewGRPCLogger(logger Logger) *GRPCLogger {
	return &GRPCLogger{Logger: logger}
}

// LogCall logs gRPC call details
func (l *GRPCLogger) LogCall(ctx context.Context, method string, duration time.Duration, err error) {
	fields := map[string]interface{}{
		"grpc_method": method,
		"duration":    duration.Milliseconds(),
	}

	contextLogger := l.Logger.WithContext(ctx)

	if err != nil {
		fields["error"] = err.Error()
		contextLogger.Error("gRPC call failed", fields)
	} else {
		contextLogger.Info("gRPC call completed", fields)
	}
}