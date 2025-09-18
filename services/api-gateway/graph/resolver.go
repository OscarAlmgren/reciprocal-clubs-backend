package graph

import (
	"reciprocal-clubs-backend/pkg/shared/auth"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/messaging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/api-gateway/internal/clients"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	logger       logging.Logger
	monitor      *monitoring.Monitor
	authProvider *auth.JWTProvider
	messageBus   messaging.MessageBus
	clients      *clients.ServiceClients
}

// NewResolver creates a new resolver with dependencies
func NewResolver(
	logger logging.Logger,
	monitor *monitoring.Monitor,
	authProvider *auth.JWTProvider,
	messageBus messaging.MessageBus,
	clients *clients.ServiceClients,
) *Resolver {
	return &Resolver{
		logger:       logger,
		monitor:      monitor,
		authProvider: authProvider,
		messageBus:   messageBus,
		clients:      clients,
	}
}
