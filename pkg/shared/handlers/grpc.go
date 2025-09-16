package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"reciprocal-clubs-backend/pkg/shared/errors"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCHandler provides common gRPC handling utilities
type GRPCHandler struct {
	logger  logging.Logger
	monitor *monitoring.Monitor
}

// NewGRPCHandler creates a new gRPC handler
func NewGRPCHandler(logger logging.Logger, monitor *monitoring.Monitor) *GRPCHandler {
	return &GRPCHandler{
		logger:  logger,
		monitor: monitor,
	}
}

// UnaryServerInterceptor returns a server interceptor for unary calls
func (h *GRPCHandler) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Add request ID to context
		requestID := h.generateRequestID()
		ctx = context.WithValue(ctx, "request_id", requestID)
		ctx = logging.ContextWithCorrelationID(ctx, requestID)

		// Log incoming request
		h.logger.WithContext(ctx).Info("gRPC request started", map[string]interface{}{
			"method":     info.FullMethod,
			"request_id": requestID,
		})

		// Call the handler
		resp, err := handler(ctx, req)

		duration := time.Since(start)

		// Record metrics
		status := "success"
		if err != nil {
			status = "error"
		}

		if h.monitor != nil {
			h.monitor.RecordGRPCRequest(info.FullMethod, status, duration)
		}

		// Log response
		fields := map[string]interface{}{
			"method":      info.FullMethod,
			"duration_ms": duration.Milliseconds(),
			"status":      status,
		}

		if err != nil {
			fields["error"] = err.Error()
			h.logger.WithContext(ctx).Error("gRPC request completed with error", fields)
		} else {
			h.logger.WithContext(ctx).Info("gRPC request completed", fields)
		}

		return resp, err
	}
}

// StreamServerInterceptor returns a server interceptor for streaming calls
func (h *GRPCHandler) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		ctx := ss.Context()

		// Add request ID to context
		requestID := h.generateRequestID()
		ctx = context.WithValue(ctx, "request_id", requestID)
		ctx = logging.ContextWithCorrelationID(ctx, requestID)

		// Create wrapped stream with updated context
		wrapped := &wrappedServerStream{ServerStream: ss, ctx: ctx}

		// Log incoming stream request
		h.logger.WithContext(ctx).Info("gRPC stream request started", map[string]interface{}{
			"method":     info.FullMethod,
			"request_id": requestID,
		})

		// Call the handler
		err := handler(srv, wrapped)

		duration := time.Since(start)

		// Record metrics
		status := "success"
		if err != nil {
			status = "error"
		}

		if h.monitor != nil {
			h.monitor.RecordGRPCRequest(info.FullMethod, status, duration)
		}

		// Log response
		fields := map[string]interface{}{
			"method":      info.FullMethod,
			"duration_ms": duration.Milliseconds(),
			"status":      status,
		}

		if err != nil {
			fields["error"] = err.Error()
			h.logger.WithContext(ctx).Error("gRPC stream request completed with error", fields)
		} else {
			h.logger.WithContext(ctx).Info("gRPC stream request completed", fields)
		}

		return err
	}
}

// RecoveryInterceptor handles panics in gRPC handlers
func (h *GRPCHandler) RecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		defer func() {
			if r := recover(); r != nil {
				h.logger.WithContext(ctx).Error("Panic recovered in gRPC handler", map[string]interface{}{
					"error":  fmt.Sprintf("%v", r),
					"method": info.FullMethod,
				})
			}
		}()

		return handler(ctx, req)
	}
}

// ValidationInterceptor validates incoming requests
func (h *GRPCHandler) ValidationInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Check if request implements a Validate method
		if validator, ok := req.(interface{ Validate() error }); ok {
			if err := validator.Validate(); err != nil {
				h.logger.WithContext(ctx).Warn("Request validation failed", map[string]interface{}{
					"error":  err.Error(),
					"method": info.FullMethod,
				})
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
		}

		return handler(ctx, req)
	}
}

// HandleError converts application errors to gRPC status errors
func (h *GRPCHandler) HandleError(err error) error {
	if err == nil {
		return nil
	}

	if appErr, ok := err.(*errors.AppError); ok {
		code := h.getGRPCCodeFromError(appErr.Code)

		// Create status with details
		st := status.New(code, appErr.Message)

		// Add error details if available
		if len(appErr.Fields) > 0 {
			// In a real implementation, you might use status.WithDetails
			// to add structured error information
			h.logger.Debug("Error details", appErr.Fields)
		}

		return st.Err()
	}

	// Default to internal error
	return status.Error(codes.Internal, err.Error())
}

// getGRPCCodeFromError converts application error codes to gRPC codes
func (h *GRPCHandler) getGRPCCodeFromError(code errors.ErrorCode) codes.Code {
	switch code {
	case errors.ErrNotFound:
		return codes.NotFound
	case errors.ErrInvalidInput:
		return codes.InvalidArgument
	case errors.ErrUnauthorized:
		return codes.Unauthenticated
	case errors.ErrForbidden:
		return codes.PermissionDenied
	case errors.ErrConflict:
		return codes.AlreadyExists
	case errors.ErrTimeout:
		return codes.DeadlineExceeded
	case errors.ErrUnavailable:
		return codes.Unavailable
	default:
		return codes.Internal
	}
}

// generateRequestID generates a unique request ID
func (h *GRPCHandler) generateRequestID() string {
	return fmt.Sprintf("grpc_%d", time.Now().UnixNano())
}

// wrappedServerStream wraps grpc.ServerStream with a custom context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the custom context
func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// BaseService provides common functionality for gRPC services
type BaseService struct {
	logger  logging.Logger
	monitor *monitoring.Monitor
	handler *GRPCHandler
}

// NewBaseService creates a new base service
func NewBaseService(logger logging.Logger, monitor *monitoring.Monitor) *BaseService {
	return &BaseService{
		logger:  logger,
		monitor: monitor,
		handler: NewGRPCHandler(logger, monitor),
	}
}

// ValidateRequest validates a request message
func (s *BaseService) ValidateRequest(req interface{}) error {
	if validator, ok := req.(interface{ Validate() error }); ok {
		return validator.Validate()
	}
	return nil
}

// LogRequest logs an incoming request
func (s *BaseService) LogRequest(ctx context.Context, method string, req interface{}) {
	s.logger.WithContext(ctx).Info("gRPC request received", map[string]interface{}{
		"method": method,
		"type":   fmt.Sprintf("%T", req),
	})
}

// LogResponse logs a response
func (s *BaseService) LogResponse(ctx context.Context, method string, resp interface{}, err error) {
	fields := map[string]interface{}{
		"method": method,
	}

	if err != nil {
		fields["error"] = err.Error()
		s.logger.WithContext(ctx).Error("gRPC request failed", fields)
	} else {
		fields["type"] = fmt.Sprintf("%T", resp)
		s.logger.WithContext(ctx).Info("gRPC request successful", fields)
	}
}

// HandleError handles errors in gRPC handlers
func (s *BaseService) HandleError(err error) error {
	return s.handler.HandleError(err)
}

// GetUserID extracts user ID from context
func (s *BaseService) GetUserID(ctx context.Context) (uint, error) {
	if userID := logging.GetUserID(ctx); userID != nil {
		if id, ok := userID.(uint); ok {
			return id, nil
		}
		if id, ok := userID.(float64); ok {
			return uint(id), nil
		}
		if id, ok := userID.(int); ok {
			return uint(id), nil
		}
	}
	return 0, errors.Unauthorized("User ID not found in context", nil)
}

// GetClubID extracts club ID from context
func (s *BaseService) GetClubID(ctx context.Context) (uint, error) {
	if clubID := logging.GetClubID(ctx); clubID != nil {
		if id, ok := clubID.(uint); ok {
			return id, nil
		}
		if id, ok := clubID.(float64); ok {
			return uint(id), nil
		}
		if id, ok := clubID.(int); ok {
			return uint(id), nil
		}
	}
	return 0, errors.Unauthorized("Club ID not found in context", nil)
}

// HealthCheckService implements basic health check functionality
type HealthCheckService struct {
	*BaseService
}

// NewHealthCheckService creates a new health check service
func NewHealthCheckService(logger logging.Logger, monitor *monitoring.Monitor) *HealthCheckService {
	return &HealthCheckService{
		BaseService: NewBaseService(logger, monitor),
	}
}

// Common validation helpers

// ValidateID validates that an ID is greater than 0
func ValidateID(id uint32, fieldName string) error {
	if id == 0 {
		return errors.InvalidInput(fmt.Sprintf("%s is required", fieldName), map[string]interface{}{
			"field": fieldName,
			"value": id,
		}, nil)
	}
	return nil
}

// ValidateString validates that a string is not empty
func ValidateString(value, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return errors.InvalidInput(fmt.Sprintf("%s is required", fieldName), map[string]interface{}{
			"field": fieldName,
		}, nil)
	}
	return nil
}

// ValidateEmail validates email format
func ValidateEmail(email string) error {
	if err := ValidateString(email, "email"); err != nil {
		return err
	}

	// Basic email validation - in production, use a proper email validation library
	if !contains(email, "@") || !contains(email, ".") {
		return errors.InvalidInput("Invalid email format", map[string]interface{}{
			"email": email,
		}, nil)
	}

	return nil
}

// ValidatePagination validates pagination parameters
func ValidatePagination(page, pageSize int32) error {
	if page < 1 {
		return errors.InvalidInput("Page must be greater than 0", map[string]interface{}{
			"page": page,
		}, nil)
	}

	if pageSize < 1 {
		return errors.InvalidInput("Page size must be greater than 0", map[string]interface{}{
			"page_size": pageSize,
		}, nil)
	}

	if pageSize > 100 {
		return errors.InvalidInput("Page size must not exceed 100", map[string]interface{}{
			"page_size": pageSize,
			"max_size":  100,
		}, nil)
	}

	return nil
}

// Helper functions

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && findSubstring(s, substr) >= 0))
}

// findSubstring finds the first occurrence of substr in s
func findSubstring(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(s) {
		return -1
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// ServerOptions provides common server options
type ServerOptions struct {
	MaxRecvMsgSize int
	MaxSendMsgSize int
}

// DefaultServerOptions returns default gRPC server options
func DefaultServerOptions() ServerOptions {
	return ServerOptions{
		MaxRecvMsgSize: 4 * 1024 * 1024, // 4MB
		MaxSendMsgSize: 4 * 1024 * 1024, // 4MB
	}
}

// ApplyServerOptions applies server options to gRPC server
func (opts ServerOptions) ApplyServerOptions() []grpc.ServerOption {
	var options []grpc.ServerOption

	if opts.MaxRecvMsgSize > 0 {
		options = append(options, grpc.MaxRecvMsgSize(opts.MaxRecvMsgSize))
	}

	if opts.MaxSendMsgSize > 0 {
		options = append(options, grpc.MaxSendMsgSize(opts.MaxSendMsgSize))
	}

	return options
}

// GetDefaultInterceptors returns default interceptors for gRPC server
func (h *GRPCHandler) GetDefaultInterceptors() ([]grpc.UnaryServerInterceptor, []grpc.StreamServerInterceptor) {
	unary := []grpc.UnaryServerInterceptor{
		h.RecoveryInterceptor(),
		h.ValidationInterceptor(),
		h.UnaryServerInterceptor(),
	}

	stream := []grpc.StreamServerInterceptor{
		h.StreamServerInterceptor(),
	}

	return unary, stream
}
