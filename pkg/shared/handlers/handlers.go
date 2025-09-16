package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"reciprocal-clubs-backend/pkg/shared/auth"
	"reciprocal-clubs-backend/pkg/shared/errors"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"

	"google.golang.org/grpc"
)

// This file provides convenient access to all handler types and utilities

// HTTP Handler Types
type (
	// HTTPHandlerInterface defines the interface for HTTP handlers
	HTTPHandlerInterface interface {
		WriteResponse(w http.ResponseWriter, r *http.Request, statusCode int, data interface{})
		WriteError(w http.ResponseWriter, r *http.Request, err error)
		ParseJSONBody(r *http.Request, target interface{}) error
	}

	// AuthHandlerInterface defines the interface for authentication handlers
	AuthHandlerInterface interface {
		Login(w http.ResponseWriter, r *http.Request)
		Register(w http.ResponseWriter, r *http.Request)
		RefreshToken(w http.ResponseWriter, r *http.Request)
		Logout(w http.ResponseWriter, r *http.Request)
		Me(w http.ResponseWriter, r *http.Request)
	}
)

// gRPC Handler Types
type (
	// GRPCHandlerInterface defines the interface for gRPC handlers
	GRPCHandlerInterface interface {
		HandleError(err error) error
	}
)

// Middleware Types
type (
	// MiddlewareFunc represents a middleware function
	MiddlewareFunc func(http.Handler) http.Handler

	// InterceptorFunc represents a gRPC interceptor function
	UnaryInterceptorFunc  = grpc.UnaryServerInterceptor
	StreamInterceptorFunc = grpc.StreamServerInterceptor
)

// Common response types
type (
	// APIResponse represents a standard API response
	APIResponse = StandardResponse

	// ErrorDetails represents error details in responses
	ErrorDetails = ErrorResponse

	// PaginationInfo represents pagination metadata
	PaginationInfo = PaginationMeta
)

// Validation types
type (
	// ValidationError represents a validation error
	ValidationError = errors.AppError

	// FieldError represents a field-specific error
	FieldError struct {
		Field   string      `json:"field"`
		Value   interface{} `json:"value,omitempty"`
		Message string      `json:"message"`
	}
)

// Configuration types
type (
	// CORSConfiguration represents CORS settings
	CORSConfiguration = CORSConfig

	// RateLimitConfiguration represents rate limiting settings
	RateLimitConfiguration = RateLimitConfig
)

// Factory functions for easy creation

// NewHTTPHandlerChain creates a new middleware chain with common middleware
func NewHTTPHandlerChain(logger logging.Logger, monitor *monitoring.Monitor) *MiddlewareChain {
	chain := NewMiddlewareChain()
	chain.Use(RecoveryMiddleware(logger))
	chain.Use(SecurityHeadersMiddleware())
	chain.Use(RequestIDMiddleware())
	chain.Use(LoggingMiddleware(logger))
	if monitor != nil {
		chain.Use(MetricsMiddleware(monitor))
	}
	return chain
}

// NewCORSMiddleware creates CORS middleware with default configuration
func NewCORSMiddleware() MiddlewareFunc {
	return CORSMiddleware(DefaultCORSConfig())
}

// NewRateLimitMiddleware creates rate limiting middleware with default configuration
func NewRateLimitMiddleware(logger logging.Logger) MiddlewareFunc {
	return RateLimitMiddleware(logger, DefaultRateLimitConfig())
}

// NewAuthMiddleware creates a complete authentication handler
func NewAuthMiddleware(logger logging.Logger, authProvider auth.AuthProvider) *AuthHandler {
	return NewAuthHandler(logger, authProvider)
}

// NewGRPCMiddleware creates gRPC interceptors with common functionality
func NewGRPCMiddleware(logger logging.Logger, monitor *monitoring.Monitor) ([]grpc.UnaryServerInterceptor, []grpc.StreamServerInterceptor) {
	handler := NewGRPCHandler(logger, monitor)
	return handler.GetDefaultInterceptors()
}

// Utility functions

// ValidateHTTPRequest validates common HTTP request parameters
func ValidateHTTPRequest(r *http.Request, requiredFields ...string) error {
	// Basic validation logic
	if r.Body == nil && len(requiredFields) > 0 {
		return errors.InvalidInput("Request body is required", nil, nil)
	}

	// Add more validation as needed
	return nil
}

// CreateErrorResponse creates a standardized error response
func CreateErrorResponse(err error) *ErrorResponse {
	if appErr, ok := err.(*errors.AppError); ok {
		return &ErrorResponse{
			Code:    string(appErr.Code),
			Message: appErr.Message,
			Details: appErr.Fields,
		}
	}

	return &ErrorResponse{
		Code:    "INTERNAL_ERROR",
		Message: err.Error(),
	}
}

// CreateSuccessResponse creates a standardized success response
func CreateSuccessResponse(data interface{}, meta *ResponseMeta) *StandardResponse {
	return &StandardResponse{
		Success:   true,
		Data:      data,
		Meta:      meta,
		Timestamp: time.Now().UTC(),
	}
}

// Helper constants for common HTTP status messages
const (
	MessageSuccess            = "Request completed successfully"
	MessageCreated            = "Resource created successfully"
	MessageUpdated            = "Resource updated successfully"
	MessageDeleted            = "Resource deleted successfully"
	MessageNotFound           = "Resource not found"
	MessageUnauthorized       = "Authentication required"
	MessageForbidden          = "Access denied"
	MessageBadRequest         = "Invalid request"
	MessageInternalError      = "Internal server error"
	MessageServiceUnavailable = "Service temporarily unavailable"
)

// Common field validation functions
func ValidateRequiredField(value, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return errors.InvalidInput(fmt.Sprintf("%s is required", fieldName), map[string]interface{}{
			"field": fieldName,
		}, nil)
	}
	return nil
}

func ValidateIDField(id uint, fieldName string) error {
	if id == 0 {
		return errors.InvalidInput(fmt.Sprintf("%s must be greater than 0", fieldName), map[string]interface{}{
			"field": fieldName,
			"value": id,
		}, nil)
	}
	return nil
}

func ValidateEnumField(value string, validValues []string, fieldName string) error {
	for _, valid := range validValues {
		if value == valid {
			return nil
		}
	}

	return errors.InvalidInput(fmt.Sprintf("Invalid %s", fieldName), map[string]interface{}{
		"field":        fieldName,
		"value":        value,
		"valid_values": validValues,
	}, nil)
}

// Response helper functions

// RespondWithJSON writes a JSON response with the given status code
func RespondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

// RespondWithError writes an error response
func RespondWithError(w http.ResponseWriter, statusCode int, message string) {
	RespondWithJSON(w, statusCode, map[string]string{"error": message})
}

// ExtractBearerToken extracts bearer token from Authorization header
func ExtractBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.Unauthorized("Authorization header required", nil)
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", errors.Unauthorized("Invalid authorization header format", nil)
	}

	return parts[1], nil
}

// GetRequestID extracts request ID from context
func GetRequestID(ctx context.Context) string {
	if id := ctx.Value("request_id"); id != nil {
		if requestID, ok := id.(string); ok {
			return requestID
		}
	}
	return ""
}
