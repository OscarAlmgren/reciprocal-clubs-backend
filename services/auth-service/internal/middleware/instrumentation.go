package middleware

import (
	"context"
	"strconv"
	"time"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/auth-service/internal/metrics"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// InstrumentationMiddleware provides comprehensive instrumentation for gRPC services
type InstrumentationMiddleware struct {
	monitor     *monitoring.Monitor
	authMetrics *metrics.AuthMetrics
	logger      logging.Logger
}

// NewInstrumentationMiddleware creates a new instrumentation middleware
func NewInstrumentationMiddleware(monitor *monitoring.Monitor, authMetrics *metrics.AuthMetrics, logger logging.Logger) *InstrumentationMiddleware {
	return &InstrumentationMiddleware{
		monitor:     monitor,
		authMetrics: authMetrics,
		logger:      logger,
	}
}

// UnaryServerInterceptor returns a gRPC unary server interceptor for instrumentation
func (i *InstrumentationMiddleware) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()

		// Increment concurrent requests
		i.authMetrics.IncrementConcurrentRequests()
		defer i.authMetrics.DecrementConcurrentRequests()

		// Extract metadata from context
		correlationID := logging.GetCorrelationID(ctx)
		if correlationID == \"\" {
			correlationID = logging.GenerateCorrelationID()
			ctx = logging.SetCorrelationID(ctx, correlationID)
		}

		// Add request context to logger
		requestLogger := i.logger.WithContext(ctx).With(map[string]interface{}{
			\"method\":         info.FullMethod,
			\"correlation_id\": correlationID,
			\"start_time\":     startTime,
		})

		requestLogger.Info(\"gRPC request started\", map[string]interface{}{
			\"request_size\": estimateRequestSize(req),
		})

		// Execute the handler
		resp, err := handler(ctx, req)

		// Calculate duration and determine status
		duration := time.Since(startTime)
		grpcStatus := status.Code(err)
		statusStr := grpcStatus.String()

		// Record metrics
		i.monitor.RecordGRPCRequest(info.FullMethod, statusStr, duration)

		// Record Auth Service specific metrics based on method
		i.recordMethodSpecificMetrics(info.FullMethod, ctx, duration, err, req, resp)

		// Log request completion
		logFields := map[string]interface{}{
			\"duration_ms\":    duration.Milliseconds(),
			\"status\":         statusStr,
			\"status_code\":    int(grpcStatus),
			\"response_size\":  estimateResponseSize(resp),
		}

		if err != nil {
			logFields[\"error\"] = err.Error()
			requestLogger.Error(\"gRPC request failed\", logFields)
		} else {
			requestLogger.Info(\"gRPC request completed\", logFields)
		}

		return resp, err
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor for instrumentation
func (i *InstrumentationMiddleware) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		startTime := time.Now()

		// Increment concurrent requests
		i.authMetrics.IncrementConcurrentRequests()
		defer i.authMetrics.DecrementConcurrentRequests()

		ctx := stream.Context()
		correlationID := logging.GetCorrelationID(ctx)
		if correlationID == \"\" {
			correlationID = logging.GenerateCorrelationID()
			ctx = logging.SetCorrelationID(ctx, correlationID)
		}

		requestLogger := i.logger.WithContext(ctx).With(map[string]interface{}{
			\"method\":         info.FullMethod,
			\"correlation_id\": correlationID,
			\"is_stream\":      true,
		})

		requestLogger.Info(\"gRPC stream started\", nil)

		// Execute the handler
		err := handler(srv, stream)

		duration := time.Since(startTime)
		grpcStatus := status.Code(err)
		statusStr := grpcStatus.String()

		// Record metrics
		i.monitor.RecordGRPCRequest(info.FullMethod, statusStr, duration)

		// Log completion
		logFields := map[string]interface{}{
			\"duration_ms\": duration.Milliseconds(),
			\"status\":      statusStr,
			\"status_code\": int(grpcStatus),
		}

		if err != nil {
			logFields[\"error\"] = err.Error()
			requestLogger.Error(\"gRPC stream failed\", logFields)
		} else {
			requestLogger.Info(\"gRPC stream completed\", logFields)
		}

		return err
	}
}

// recordMethodSpecificMetrics records metrics specific to Auth Service methods
func (i *InstrumentationMiddleware) recordMethodSpecificMetrics(method string, ctx context.Context, duration time.Duration, err error, req, resp interface{}) {
	clubID := extractClubID(ctx, req)
	result := \"success\"
	if err != nil {
		result = \"error\"
	}

	switch method {
	case \"/auth.AuthService/RegisterUser\":
		i.authMetrics.RecordUserRegistration(clubID, \"grpc\", result)
		i.authMetrics.RecordAuthDuration(\"register\", clubID, result, duration)

	case \"/auth.AuthService/InitiatePasskeyLogin\":
		i.authMetrics.RecordPasskeyOperation(\"initiate_login\", clubID, result, extractUserID(ctx, req))
		i.authMetrics.RecordPasskeyDuration(\"initiate_login\", clubID, duration)

	case \"/auth.AuthService/CompletePasskeyLogin\":
		i.authMetrics.RecordPasskeyOperation(\"complete_login\", clubID, result, extractUserID(ctx, req))
		i.authMetrics.RecordPasskeyDuration(\"complete_login\", clubID, duration)
		i.authMetrics.RecordAuthDuration(\"passkey_login\", clubID, result, duration)

		if err == nil {
			i.authMetrics.RecordAuthSuccess(\"passkey\", clubID, \"member\")
		} else {
			i.authMetrics.RecordAuthFailure(\"passkey\", clubID, getErrorReason(err), extractUserID(ctx, req))
		}

	case \"/auth.AuthService/InitiatePasskeyRegistration\":
		i.authMetrics.RecordPasskeyOperation(\"initiate_registration\", clubID, result, extractUserID(ctx, req))
		i.authMetrics.RecordPasskeyDuration(\"initiate_registration\", clubID, duration)

	case \"/auth.AuthService/CompletePasskeyRegistration\":
		i.authMetrics.RecordPasskeyOperation(\"complete_registration\", clubID, result, extractUserID(ctx, req))
		i.authMetrics.RecordPasskeyDuration(\"complete_registration\", clubID, duration)

	case \"/auth.AuthService/ValidateSession\":
		i.authMetrics.RecordAuthDuration(\"validate_session\", clubID, result, duration)

	case \"/auth.AuthService/Logout\":
		userID := extractUserID(ctx, req)
		if err == nil {
			// Calculate session duration (would need session start time)
			i.authMetrics.RecordSessionDuration(clubID, \"standard\", \"logout\", duration)
		}

	case \"/auth.AuthService/AssignRole\":
		i.authMetrics.RecordRoleAssignment(clubID, extractRoleName(req), \"assign\", extractGrantedBy(ctx))

	case \"/auth.AuthService/RemoveRole\":
		i.authMetrics.RecordRoleAssignment(clubID, extractRoleName(req), \"remove\", extractGrantedBy(ctx))

	case \"/auth.AuthService/GetUserPermissions\":
		i.authMetrics.RecordPermissionCheck(clubID, \"*\", \"user_permissions\", result)

	case \"/auth.AuthService/GetAuditLogs\":
		i.authMetrics.RecordAuditEvent(clubID, \"query\", \"audit_logs\", extractUserID(ctx, req), result)
	}
}

// Helper functions to extract metadata from requests

func extractClubID(ctx context.Context, req interface{}) string {
	// Try to extract club_id from various request types
	// This would need to be implemented based on your specific request structures
	if clubIDInterface := ctx.Value(\"club_id\"); clubIDInterface != nil {
		if clubID, ok := clubIDInterface.(string); ok {
			return clubID
		}
		if clubID, ok := clubIDInterface.(uint); ok {
			return strconv.FormatUint(uint64(clubID), 10)
		}
	}
	return \"unknown\"
}

func extractUserID(ctx context.Context, req interface{}) string {
	// Try to extract user_id from context or request
	if userIDInterface := ctx.Value(\"user_id\"); userIDInterface != nil {
		if userID, ok := userIDInterface.(string); ok {
			return userID
		}
		if userID, ok := userIDInterface.(uint); ok {
			return strconv.FormatUint(uint64(userID), 10)
		}
	}
	return \"unknown\"
}

func extractRoleName(req interface{}) string {
	// Extract role name from role assignment requests
	// Implementation depends on your specific request structures
	return \"unknown\"
}

func extractGrantedBy(ctx context.Context) string {
	// Extract who granted the role from context
	if grantedByInterface := ctx.Value(\"granted_by\"); grantedByInterface != nil {
		if grantedBy, ok := grantedByInterface.(string); ok {
			return grantedBy
		}
	}
	return \"system\"
}

func getErrorReason(err error) string {
	if err == nil {
		return \"\"
	}

	// Map gRPC status codes to reasons
	grpcStatus := status.Code(err)
	switch grpcStatus {
	case codes.Unauthenticated:
		return \"invalid_credentials\"
	case codes.PermissionDenied:
		return \"access_denied\"
	case codes.NotFound:
		return \"user_not_found\"
	case codes.InvalidArgument:
		return \"invalid_request\"
	case codes.FailedPrecondition:
		return \"precondition_failed\"
	case codes.ResourceExhausted:
		return \"rate_limited\"
	case codes.Internal:
		return \"internal_error\"
	case codes.Unavailable:
		return \"service_unavailable\"
	default:
		return \"unknown_error\"
	}
}

func estimateRequestSize(req interface{}) int {
	// Simple estimation - in production you might want more accurate sizing
	if req == nil {
		return 0
	}
	// This is a rough estimate - implement proper sizing based on your needs
	return 100
}

func estimateResponseSize(resp interface{}) int {
	// Simple estimation - in production you might want more accurate sizing
	if resp == nil {
		return 0
	}
	// This is a rough estimate - implement proper sizing based on your needs
	return 200
}

// HealthCheckMetrics records health check related metrics
func (i *InstrumentationMiddleware) RecordHealthCheck(component string, healthy bool, duration time.Duration) {
	status := \"healthy\"
	if !healthy {
		status = \"unhealthy\"
	}

	i.monitor.GetMetrics().HealthStatus.WithLabelValues(component, status).Set(1)

	i.logger.Info(\"Health check completed\", map[string]interface{}{
		\"component\":   component,
		\"healthy\":     healthy,
		\"duration_ms\": duration.Milliseconds(),
	})
}

// RecordStartup records service startup metrics
func (i *InstrumentationMiddleware) RecordStartup() {
	i.monitor.GetMetrics().ServiceUptime.Inc()
	i.logger.Info(\"Auth service startup recorded\", nil)
}