package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"reciprocal-clubs-backend/pkg/shared/auth"
	"reciprocal-clubs-backend/pkg/shared/config"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/messaging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/api-gateway/graph"
	"reciprocal-clubs-backend/services/api-gateway/graph/generated"
	"reciprocal-clubs-backend/services/api-gateway/internal/clients"
	"reciprocal-clubs-backend/services/api-gateway/internal/middleware"
	"reciprocal-clubs-backend/services/api-gateway/internal/metrics"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server represents the HTTP server
type Server struct {
	config         *config.Config
	logger         logging.Logger
	monitor        *monitoring.Monitor
	authProvider   *auth.JWTProvider
	messageBus     messaging.MessageBus
	clients        *clients.ServiceClients
	router         *mux.Router
	gatewayMetrics *metrics.APIGatewayMetrics
}

// NewServer creates a new HTTP server instance
func NewServer(cfg *config.Config, logger logging.Logger, monitor *monitoring.Monitor) (*Server, error) {
	// Initialize auth provider
	authProvider := auth.NewJWTProvider(&cfg.Auth, logger)

	// Initialize message bus
	messageBus, err := messaging.NewNATSMessageBus(&cfg.NATS, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create message bus: %w", err)
	}

	// Initialize service clients
	serviceClients, err := clients.NewServiceClients(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create service clients: %w", err)
	}

	// Initialize gateway metrics
	gatewayMetrics := metrics.NewAPIGatewayMetrics(monitor, logger)

	server := &Server{
		config:         cfg,
		logger:         logger,
		monitor:        monitor,
		authProvider:   authProvider,
		messageBus:     messageBus,
		clients:        serviceClients,
		router:         mux.NewRouter(),
		gatewayMetrics: gatewayMetrics,
	}

	// Register health checks
	server.registerHealthChecks()

	// Setup routes
	server.setupRoutes()

	return server, nil
}

// Handler returns the HTTP handler
func (s *Server) Handler() http.Handler {
	return s.router
}

// Close closes server resources
func (s *Server) Close() error {
	if s.messageBus != nil {
		if err := s.messageBus.Close(); err != nil {
			s.logger.Error("Error closing message bus", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	if s.clients != nil {
		if err := s.clients.Close(); err != nil {
			s.logger.Error("Error closing service clients", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	return nil
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// Health check endpoints
	s.router.HandleFunc("/health", s.monitor.HealthCheckHandler()).Methods("GET")
	s.router.HandleFunc("/ready", s.monitor.ReadinessCheckHandler()).Methods("GET")
	s.router.HandleFunc("/live", s.handleLiveness).Methods("GET")

	// Metrics endpoint
	s.router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// GraphQL endpoints
	s.setupGraphQLRoutes()

	// REST API endpoints
	s.setupRESTRoutes()

	// Apply middleware in order (order matters!)
	s.router.Use(middleware.SecurityHeadersMiddleware())
	s.router.Use(middleware.RequestIDMiddleware())
	s.router.Use(middleware.RequestSizeLimitMiddleware(10*1024*1024, s.logger)) // 10MB limit
	s.router.Use(middleware.RequestTimeoutMiddleware(s.logger))
	s.router.Use(s.createEnhancedLoggingMiddleware())
	s.router.Use(s.createEnhancedMetricsMiddleware())
	s.router.Use(s.createAdvancedRateLimitMiddleware())
}

// setupGraphQLRoutes configures GraphQL endpoints
func (s *Server) setupGraphQLRoutes() {
	// Create GraphQL server
	graphqlServer := s.createGraphQLServer()

	// GraphQL endpoint
	s.router.Handle("/graphql", graphqlServer).Methods("POST", "GET")

	// GraphQL playground (development only)
	if s.config.Service.Environment != "production" {
		s.router.Handle("/playground", playground.Handler("GraphQL Playground", "/graphql")).Methods("GET")
		s.logger.Info("GraphQL playground enabled at /playground", nil)
	}
}

// createGraphQLServer creates and configures the GraphQL server
func (s *Server) createGraphQLServer() http.Handler {
	// Create resolver with dependencies
	resolver := graph.NewResolver(
		s.logger,
		s.monitor,
		s.authProvider,
		s.messageBus,
		s.clients,
	)

	// Create executable schema
	schema := generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	})

	// Create GraphQL handler
	srv := handler.New(schema)

	// Configure transports
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})

	// WebSocket transport for subscriptions
	srv.AddTransport(&transport.Websocket{
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// In production, implement proper origin checking
				return true
			},
		},
		KeepAlivePingInterval: 10 * time.Second,
	})

	// Add extensions
	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New(100),
	})

	// Add complexity limiting
	srv.Use(extension.FixedComplexityLimit(300))

	// Wrap with authentication middleware for protected operations
	return middleware.GraphQLAuthMiddleware(s.authProvider, s.logger)(srv)
}

// setupRESTRoutes configures comprehensive REST API endpoints
func (s *Server) setupRESTRoutes() {
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Authentication endpoints (no auth required)
	auth := api.PathPrefix("/auth").Subrouter()
	auth.HandleFunc("/login", s.handleLogin).Methods("POST")
	auth.HandleFunc("/register", s.handleRegister).Methods("POST")
	auth.HandleFunc("/refresh", s.handleRefresh).Methods("POST")
	auth.HandleFunc("/passkey/initiate", s.handleInitiatePasskey).Methods("POST")
	auth.HandleFunc("/passkey/complete", s.handleCompletePasskey).Methods("POST")

	// Protected endpoints
	protected := api.PathPrefix("").Subrouter()
	protected.Use(s.authProvider.Middleware())

	// User management endpoints
	protected.HandleFunc("/auth/logout", s.handleLogout).Methods("POST")
	protected.HandleFunc("/auth/me", s.handleMe).Methods("GET")
	protected.HandleFunc("/users/{clubId}/{userId}", s.handleGetUser).Methods("GET")
	protected.HandleFunc("/users/{clubId}/{userId}", s.handleUpdateUser).Methods("PUT")
	protected.HandleFunc("/users/{clubId}/{userId}", s.handleDeleteUser).Methods("DELETE")

	// Member management endpoints
	protected.HandleFunc("/members", s.handleCreateMember).Methods("POST")
	protected.HandleFunc("/members/{clubId}", s.handleListMembers).Methods("GET")
	protected.HandleFunc("/members/{clubId}/{memberId}", s.handleGetMember).Methods("GET")
	protected.HandleFunc("/members/{clubId}/{memberId}", s.handleUpdateMember).Methods("PUT")
	protected.HandleFunc("/members/{clubId}/{memberId}", s.handleDeleteMember).Methods("DELETE")
	protected.HandleFunc("/members/{clubId}/search", s.handleSearchMembers).Methods("GET")
	protected.HandleFunc("/members/{clubId}/{memberId}/suspend", s.handleSuspendMember).Methods("POST")
	protected.HandleFunc("/members/{clubId}/{memberId}/activate", s.handleActivateMember).Methods("POST")
	protected.HandleFunc("/members/{clubId}/analytics", s.handleMemberAnalytics).Methods("GET")

	// Reciprocal agreement endpoints
	protected.HandleFunc("/agreements", s.handleCreateAgreement).Methods("POST")
	protected.HandleFunc("/agreements/{clubId}", s.handleListAgreements).Methods("GET")
	protected.HandleFunc("/agreements/{clubId}/{agreementId}", s.handleGetAgreement).Methods("GET")
	protected.HandleFunc("/agreements/{clubId}/{agreementId}", s.handleUpdateAgreement).Methods("PUT")

	// Visit management endpoints
	protected.HandleFunc("/visits/request", s.handleRequestVisit).Methods("POST")
	protected.HandleFunc("/visits/{clubId}/{visitId}/confirm", s.handleConfirmVisit).Methods("POST")
	protected.HandleFunc("/visits/{clubId}/{visitId}/checkin", s.handleCheckInVisit).Methods("POST")
	protected.HandleFunc("/visits/{clubId}/{visitId}/checkout", s.handleCheckOutVisit).Methods("POST")
	protected.HandleFunc("/visits/{clubId}", s.handleListVisits).Methods("GET")
	protected.HandleFunc("/visits/{clubId}/analytics", s.handleVisitAnalytics).Methods("GET")

	// Blockchain endpoints
	protected.HandleFunc("/blockchain/transactions", s.handleSubmitTransaction).Methods("POST")
	protected.HandleFunc("/blockchain/transactions/{transactionId}", s.handleGetTransaction).Methods("GET")
	protected.HandleFunc("/blockchain/transactions/{clubId}", s.handleListTransactions).Methods("GET")
	protected.HandleFunc("/blockchain/ledger/query", s.handleQueryLedger).Methods("POST")
	protected.HandleFunc("/blockchain/status/{clubId}", s.handleBlockchainStatus).Methods("GET")

	// Role and permission endpoints
	protected.HandleFunc("/roles", s.handleCreateRole).Methods("POST")
	protected.HandleFunc("/roles/{clubId}/assign", s.handleAssignRole).Methods("POST")
	protected.HandleFunc("/roles/{clubId}/remove", s.handleRemoveRole).Methods("DELETE")
	protected.HandleFunc("/permissions/check", s.handleCheckPermission).Methods("POST")
	protected.HandleFunc("/permissions/{clubId}/{userId}", s.handleGetUserPermissions).Methods("GET")

	// System status and admin endpoints
	protected.HandleFunc("/status", s.handleStatus).Methods("GET")
	protected.HandleFunc("/admin/services/connections", s.handleServiceConnections).Methods("GET")
	protected.HandleFunc("/admin/services/refresh", s.handleRefreshConnections).Methods("POST")
	protected.HandleFunc("/admin/rate-limits/{identifier}", s.handleRateLimitStatus).Methods("GET")
	protected.HandleFunc("/admin/rate-limits/{identifier}/reset", s.handleResetRateLimit).Methods("POST")
	protected.HandleFunc("/admin/circuit-breakers", s.handleCircuitBreakerStatus).Methods("GET")
	protected.HandleFunc("/admin/circuit-breakers/{name}/reset", s.handleResetCircuitBreaker).Methods("POST")
	protected.HandleFunc("/admin/analytics/requests", s.handleRequestAnalytics).Methods("GET")
	protected.HandleFunc("/admin/analytics/graphql", s.handleGraphQLAnalytics).Methods("GET")

	s.logger.Info("Comprehensive REST API routes configured", map[string]interface{}{
		"base_path":      "/api/v1",
		"total_routes":   35,
		"auth_routes":    5,
		"user_routes":    3,
		"member_routes":  8,
		"agreement_routes": 4,
		"visit_routes":   5,
		"blockchain_routes": 5,
		"role_routes":    5,
		"admin_routes":   8,
	})
}

// registerHealthChecks registers health check components
func (s *Server) registerHealthChecks() {
	// Register message bus health check
	s.monitor.RegisterHealthCheck(&messageBusHealthChecker{
		messageBus: s.messageBus,
	})

	// Register service clients health checks
	if s.clients != nil {
		s.monitor.RegisterHealthCheck(&serviceClientsHealthChecker{
			clients: s.clients,
		})
	}
}

// All REST handlers are implemented in handlers.go

// Health checkers

type messageBusHealthChecker struct {
	messageBus messaging.MessageBus
}

func (h *messageBusHealthChecker) Name() string {
	return "message_bus"
}

func (h *messageBusHealthChecker) HealthCheck(ctx context.Context) error {
	return h.messageBus.HealthCheck(ctx)
}

type serviceClientsHealthChecker struct {
	clients *clients.ServiceClients
}

func (h *serviceClientsHealthChecker) Name() string {
	return "service_clients"
}

func (h *serviceClientsHealthChecker) HealthCheck(ctx context.Context) error {
	return h.clients.HealthCheck(ctx)
}

// Enhanced middleware creators

func (s *Server) createEnhancedLoggingMiddleware() func(http.Handler) http.Handler {
	return middleware.LoggingMiddleware(s.logger)
}

func (s *Server) createEnhancedMetricsMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create response writer wrapper to capture response details
			wrapped := &enhancedResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
				responseSize:   0,
			}

			// Execute request
			next.ServeHTTP(wrapped, r)

			// Record detailed metrics
			duration := time.Since(start)
			s.gatewayMetrics.RecordHTTPRequest(
				r.Method,
				r.URL.Path,
				wrapped.statusCode,
				duration,
				wrapped.responseSize,
			)

			// Also record with shared monitor
			s.monitor.RecordHTTPRequest(r.Method, r.URL.Path, wrapped.statusCode, duration)
		})
	}
}

func (s *Server) createAdvancedRateLimitMiddleware() func(http.Handler) http.Handler {
	config := middleware.DefaultRateLimitConfig()
	config.RedisEnabled = false // TODO: Enable Redis in production
	return middleware.AdvancedRateLimitMiddleware(config, s.logger)
}

func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"alive": true, "service": "api-gateway", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`))
}

// enhancedResponseWriter captures response size along with status code
type enhancedResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	responseSize int
	written      bool
}

func (rw *enhancedResponseWriter) WriteHeader(statusCode int) {
	if !rw.written {
		rw.statusCode = statusCode
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *enhancedResponseWriter) Write(data []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(data)
	rw.responseSize += n
	return n, err
}

