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
	gatewaymonitoring "reciprocal-clubs-backend/services/api-gateway/internal/monitoring"

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
	gatewayMetrics *gatewaymonitoring.APIGatewayMetrics
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
	gatewayMetrics := gatewaymonitoring.NewAPIGatewayMetrics(logger)

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

// setupRESTRoutes configures REST API endpoints
func (s *Server) setupRESTRoutes() {
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Authentication endpoints (no auth required)
	auth := api.PathPrefix("/auth").Subrouter()
	auth.HandleFunc("/login", s.handleLogin).Methods("POST")
	auth.HandleFunc("/register", s.handleRegister).Methods("POST")
	auth.HandleFunc("/refresh", s.handleRefresh).Methods("POST")

	// Protected endpoints
	protected := api.PathPrefix("").Subrouter()
	protected.Use(s.authProvider.Middleware())

	// User endpoints
	protected.HandleFunc("/auth/logout", s.handleLogout).Methods("POST")
	protected.HandleFunc("/auth/me", s.handleMe).Methods("GET")

	// Quick status endpoints
	protected.HandleFunc("/status", s.handleStatus).Methods("GET")

	s.logger.Info("REST API routes configured", map[string]interface{}{
		"base_path": "/api/v1",
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

// REST handlers

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	// Implementation would call auth service
	// For now, return a placeholder
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte(`{"error": "not implemented yet"}`))
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	// Implementation would call auth service
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte(`{"error": "not implemented yet"}`))
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	// Implementation would refresh JWT token
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte(`{"error": "not implemented yet"}`))
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Implementation would revoke token
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true}`))
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// Return user info (implement JSON marshaling)
	w.Write([]byte(fmt.Sprintf(`{"id": "%d", "email": "%s", "username": "%s"}`, 
		user.ID, user.Email, user.Username)))
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	status := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC(),
		"service":   "api-gateway",
		"version":   s.config.Service.Version,
		"user_id":   user.ID,
		"club_id":   user.ClubID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// In a real implementation, use proper JSON marshaling
	w.Write([]byte(fmt.Sprintf(`{
		"status": "%s",
		"timestamp": "%s",
		"service": "%s",
		"version": "%s",
		"user_id": %d,
		"club_id": %d
	}`, status["status"], status["timestamp"], status["service"], 
		status["version"], status["user_id"], status["club_id"])))
}

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

