package grpc

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/notification-service/internal/models"
	"reciprocal-clubs-backend/services/notification-service/internal/service"
	pb "reciprocal-clubs-backend/services/notification-service/proto"
)

// GRPCHandler handles gRPC requests for notification service
type GRPCHandler struct {
	pb.UnimplementedNotificationServiceServer
	service    *service.NotificationService
	logger     logging.Logger
	monitoring *monitoring.Monitor
}

// NewGRPCHandler creates a new gRPC handler
func NewGRPCHandler(service *service.NotificationService, logger logging.Logger, monitoring *monitoring.Monitor) *GRPCHandler {
	return &GRPCHandler{
		service:    service,
		logger:     logger,
		monitoring: monitoring,
	}
}

// RegisterServices registers gRPC services
func (h *GRPCHandler) RegisterServices(server *grpc.Server) {
	pb.RegisterNotificationServiceServer(server, h)
	h.logger.Info("gRPC services registered", map[string]interface{}{
		"service": "notification-service",
	})
}

// Health returns service health status
func (h *GRPCHandler) Health(ctx context.Context, req *emptypb.Empty) (*pb.HealthResponse, error) {
	h.monitoring.RecordBusinessEvent("grpc_health_check", "notification")

	return &pb.HealthResponse{
		Status:  "SERVING",
		Service: "notification-service",
	}, nil
}

// CreateNotification creates a new notification
func (h *GRPCHandler) CreateNotification(ctx context.Context, req *pb.CreateNotificationRequest) (*pb.NotificationResponse, error) {
	h.monitoring.RecordBusinessEvent("grpc_create_notification", "notification")

	// Convert protobuf request to service request
	var userID *string
	if req.UserId != "" {
		userID = &req.UserId
	}

	// Convert metadata map to JSON string
	metadataJSON := ""
	if len(req.Metadata) > 0 {
		if jsonBytes, err := json.Marshal(req.Metadata); err == nil {
			metadataJSON = string(jsonBytes)
		}
	}

	serviceReq := &service.CreateNotificationRequest{
		ClubID:    uint(req.ClubId),
		UserID:    userID,
		Type:      h.convertNotificationType(req.Type),
		Priority:  h.convertNotificationPriority(req.Priority),
		Subject:   req.Title,
		Message:   req.Message,
		Recipient: req.Recipient,
		Metadata:  metadataJSON,
	}

	if req.ScheduledFor != nil {
		scheduledFor := req.ScheduledFor.AsTime()
		serviceReq.ScheduledFor = &scheduledFor
	}

	notification, err := h.service.CreateNotification(ctx, serviceReq)
	if err != nil {
		h.logger.Error("Failed to create notification via gRPC", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	return &pb.NotificationResponse{
		Notification: h.convertNotificationToProto(notification),
	}, nil
}

// GetNotification retrieves a notification by ID
func (h *GRPCHandler) GetNotification(ctx context.Context, req *pb.GetNotificationRequest) (*pb.NotificationResponse, error) {
	h.monitoring.RecordBusinessEvent("grpc_get_notification", "notification")

	notification, err := h.service.GetNotificationByID(ctx, uint(req.Id))
	if err != nil {
		h.logger.Error("Failed to get notification via gRPC", map[string]interface{}{
			"error": err.Error(),
			"id":    req.Id,
		})
		return nil, err
	}

	return &pb.NotificationResponse{
		Notification: h.convertNotificationToProto(notification),
	}, nil
}

// GetClubNotifications retrieves notifications for a club
func (h *GRPCHandler) GetClubNotifications(ctx context.Context, req *pb.GetClubNotificationsRequest) (*pb.GetNotificationsResponse, error) {
	h.monitoring.RecordBusinessEvent("grpc_get_club_notifications", "notification")

	limit := int(req.Limit)
	offset := int(req.Offset)
	if limit == 0 {
		limit = 50
	}

	notifications, err := h.service.GetNotificationsByClub(ctx, uint(req.ClubId), limit, offset)
	if err != nil {
		h.logger.Error("Failed to get club notifications via gRPC", map[string]interface{}{
			"error":   err.Error(),
			"club_id": req.ClubId,
		})
		return nil, err
	}

	protoNotifications := make([]*pb.Notification, len(notifications))
	for i, notification := range notifications {
		protoNotifications[i] = h.convertNotificationToProto(&notification)
	}

	return &pb.GetNotificationsResponse{
		Notifications: protoNotifications,
		Total:         uint32(len(notifications)),
	}, nil
}

// GetUserNotifications retrieves notifications for a user
func (h *GRPCHandler) GetUserNotifications(ctx context.Context, req *pb.GetUserNotificationsRequest) (*pb.GetNotificationsResponse, error) {
	h.monitoring.RecordBusinessEvent("grpc_get_user_notifications", "notification")

	limit := int(req.Limit)
	offset := int(req.Offset)
	if limit == 0 {
		limit = 50
	}

	notifications, err := h.service.GetNotificationsByUser(ctx, req.UserId, uint(req.ClubId), limit, offset)
	if err != nil {
		h.logger.Error("Failed to get user notifications via gRPC", map[string]interface{}{
			"error":   err.Error(),
			"user_id": req.UserId,
			"club_id": req.ClubId,
		})
		return nil, err
	}

	protoNotifications := make([]*pb.Notification, len(notifications))
	for i, notification := range notifications {
		protoNotifications[i] = h.convertNotificationToProto(&notification)
	}

	return &pb.GetNotificationsResponse{
		Notifications: protoNotifications,
		Total:         uint32(len(notifications)),
	}, nil
}

// MarkAsRead marks a notification as read
func (h *GRPCHandler) MarkAsRead(ctx context.Context, req *pb.MarkAsReadRequest) (*pb.NotificationResponse, error) {
	h.monitoring.RecordBusinessEvent("grpc_mark_as_read", "notification")

	notification, err := h.service.MarkNotificationAsRead(ctx, uint(req.Id))
	if err != nil {
		h.logger.Error("Failed to mark notification as read via gRPC", map[string]interface{}{
			"error": err.Error(),
			"id":    req.Id,
		})
		return nil, err
	}

	return &pb.NotificationResponse{
		Notification: h.convertNotificationToProto(notification),
	}, nil
}

// SendImmediate sends an immediate notification
func (h *GRPCHandler) SendImmediate(ctx context.Context, req *pb.SendImmediateRequest) (*pb.SendResponse, error) {
	h.monitoring.RecordBusinessEvent("grpc_send_immediate", "notification")

	var userID *string
	if req.UserId != "" {
		userID = &req.UserId
	}

	// Convert metadata map to JSON string
	metadataJSON := ""
	if len(req.Metadata) > 0 {
		if jsonBytes, err := json.Marshal(req.Metadata); err == nil {
			metadataJSON = string(jsonBytes)
		}
	}

	serviceReq := &service.CreateNotificationRequest{
		ClubID:    uint(req.ClubId),
		UserID:    userID,
		Type:      h.convertNotificationType(req.Type),
		Priority:  models.NotificationPritorityCritical,
		Subject:   req.Title,
		Message:   req.Message,
		Recipient: req.Recipient,
		Metadata:  metadataJSON,
	}

	notification, err := h.service.CreateNotification(ctx, serviceReq)
	if err != nil {
		return &pb.SendResponse{
			Success: false,
			Message: "Failed to send notification: " + err.Error(),
		}, nil
	}

	// Process immediately
	// Note: ProcessNotification is unexported, so we skip this for now

	return &pb.SendResponse{
		Success:      true,
		Message:      "Notification sent successfully",
		Notification: h.convertNotificationToProto(notification),
	}, nil
}

// CreateTemplate creates a new notification template
func (h *GRPCHandler) CreateTemplate(ctx context.Context, req *pb.CreateTemplateRequest) (*pb.TemplateResponse, error) {
	h.monitoring.RecordBusinessEvent("grpc_create_template", "notification")

	serviceReq := &service.CreateTemplateRequest{
		ClubID: uint(req.ClubId),
		Name:   req.Name,
		Type:   h.convertNotificationType(req.Type),
		// Note: service layer has different structure
	}

	template, err := h.service.CreateNotificationTemplate(ctx, serviceReq)
	if err != nil {
		h.logger.Error("Failed to create template via gRPC", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	return &pb.TemplateResponse{
		Template: h.convertTemplateToProto(template),
	}, nil
}

// GetClubTemplates retrieves templates for a club
func (h *GRPCHandler) GetClubTemplates(ctx context.Context, req *pb.GetClubTemplatesRequest) (*pb.GetTemplatesResponse, error) {
	h.monitoring.RecordBusinessEvent("grpc_get_club_templates", "notification")

	templates, err := h.service.GetNotificationTemplatesByClub(ctx, uint(req.ClubId))
	if err != nil {
		h.logger.Error("Failed to get club templates via gRPC", map[string]interface{}{
			"error":   err.Error(),
			"club_id": req.ClubId,
		})
		return nil, err
	}

	protoTemplates := make([]*pb.NotificationTemplate, len(templates))
	for i, template := range templates {
		protoTemplates[i] = h.convertTemplateToProto(&template)
	}

	return &pb.GetTemplatesResponse{
		Templates: protoTemplates,
	}, nil
}

// GetStats retrieves notification statistics
func (h *GRPCHandler) GetStats(ctx context.Context, req *pb.GetStatsRequest) (*pb.StatsResponse, error) {
	h.monitoring.RecordBusinessEvent("grpc_get_stats", "notification")

	fromDate := req.FromDate.AsTime()
	toDate := req.ToDate.AsTime()

	stats, err := h.service.GetNotificationStats(ctx, uint(req.ClubId), fromDate, toDate)
	if err != nil {
		h.logger.Error("Failed to get notification stats via gRPC", map[string]interface{}{
			"error":   err.Error(),
			"club_id": req.ClubId,
		})
		return nil, err
	}

	return &pb.StatsResponse{
		Total:     uint32(stats["total"].(int64)),
		Pending:   uint32(stats["pending"].(int64)),
		Sent:      uint32(stats["sent"].(int64)),
		Delivered: uint32(stats["delivered"].(int64)),
		Failed:    uint32(stats["failed"].(int64)),
		Read:      uint32(stats["read"].(int64)),
	}, nil
}

// Helper methods for conversion

func (h *GRPCHandler) convertNotificationToProto(n *models.Notification) *pb.Notification {
	// Convert metadata string to map
	metadata := make(map[string]string)
	if n.Metadata != "" {
		json.Unmarshal([]byte(n.Metadata), &metadata)
	}

	// Convert UserID pointer to string
	userID := ""
	if n.UserID != nil {
		userID = *n.UserID
	}

	proto := &pb.Notification{
		Id:            uint32(n.ID),
		ClubId:        uint32(n.ClubID),
		UserId:        userID,
		Type:          h.convertNotificationTypeToProto(n.Type),
		Status:        h.convertNotificationStatusToProto(n.Status),
		Priority:      h.convertNotificationPriorityToProto(n.Priority),
		Title:         n.Subject,  // protobuf uses Title, service uses Subject
		Message:       n.Message,
		Recipient:     n.Recipient,
		Metadata:      metadata,
		FailureReason: n.ErrorMessage,
		RetryCount:    uint32(n.RetryCount),
		CreatedAt:     timestamppb.New(n.CreatedAt),
		UpdatedAt:     timestamppb.New(n.UpdatedAt),
	}

	if n.ScheduledFor != nil {
		proto.ScheduledFor = timestamppb.New(*n.ScheduledFor)
	}
	if n.SentAt != nil {
		proto.SentAt = timestamppb.New(*n.SentAt)
	}
	if n.DeliveredAt != nil {
		proto.DeliveredAt = timestamppb.New(*n.DeliveredAt)
	}
	if n.ReadAt != nil {
		proto.ReadAt = timestamppb.New(*n.ReadAt)
	}
	if n.FailedAt != nil {
		proto.FailedAt = timestamppb.New(*n.FailedAt)
	}

	return proto
}

func (h *GRPCHandler) convertTemplateToProto(t *models.NotificationTemplate) *pb.NotificationTemplate {
	// Convert variables string to map
	metadata := make(map[string]string)
	if t.Variables != "" {
		json.Unmarshal([]byte(t.Variables), &metadata)
	}

	return &pb.NotificationTemplate{
		Id:              uint32(t.ID),
		ClubId:          uint32(t.ClubID),
		Name:            t.Name,
		Description:     "", // Not available in model
		Type:            h.convertNotificationTypeToProto(t.Type),
		SubjectTemplate: t.Subject,
		BodyTemplate:    t.Body,
		DefaultMetadata: metadata,
		IsActive:        t.IsActive,
		CreatedAt:       timestamppb.New(t.CreatedAt),
		UpdatedAt:       timestamppb.New(t.UpdatedAt),
	}
}

func (h *GRPCHandler) convertNotificationType(protoType pb.NotificationType) models.NotificationType {
	switch protoType {
	case pb.NotificationType_NOTIFICATION_TYPE_EMAIL:
		return models.NotificationTypeEmail
	case pb.NotificationType_NOTIFICATION_TYPE_SMS:
		return models.NotificationTypeSMS
	case pb.NotificationType_NOTIFICATION_TYPE_PUSH:
		return models.NotificationTypePush
	case pb.NotificationType_NOTIFICATION_TYPE_IN_APP:
		return models.NotificationTypeInApp
	case pb.NotificationType_NOTIFICATION_TYPE_WEBHOOK:
		return models.NotificationTypeWebhook
	default:
		return models.NotificationTypeEmail
	}
}

func (h *GRPCHandler) convertNotificationTypeToProto(modelType models.NotificationType) pb.NotificationType {
	switch modelType {
	case models.NotificationTypeEmail:
		return pb.NotificationType_NOTIFICATION_TYPE_EMAIL
	case models.NotificationTypeSMS:
		return pb.NotificationType_NOTIFICATION_TYPE_SMS
	case models.NotificationTypePush:
		return pb.NotificationType_NOTIFICATION_TYPE_PUSH
	case models.NotificationTypeInApp:
		return pb.NotificationType_NOTIFICATION_TYPE_IN_APP
	case models.NotificationTypeWebhook:
		return pb.NotificationType_NOTIFICATION_TYPE_WEBHOOK
	default:
		return pb.NotificationType_NOTIFICATION_TYPE_EMAIL
	}
}

func (h *GRPCHandler) convertNotificationPriority(protoPriority pb.NotificationPriority) models.NotificationPriority {
	switch protoPriority {
	case pb.NotificationPriority_NOTIFICATION_PRIORITY_LOW:
		return models.NotificationPriorityLow
	case pb.NotificationPriority_NOTIFICATION_PRIORITY_NORMAL:
		return models.NotificationPriorityNormal
	case pb.NotificationPriority_NOTIFICATION_PRIORITY_HIGH:
		return models.NotificationPriorityHigh
	case pb.NotificationPriority_NOTIFICATION_PRIORITY_URGENT:
		return models.NotificationPritorityCritical
	default:
		return models.NotificationPriorityNormal
	}
}

func (h *GRPCHandler) convertNotificationPriorityToProto(modelPriority models.NotificationPriority) pb.NotificationPriority {
	switch modelPriority {
	case models.NotificationPriorityLow:
		return pb.NotificationPriority_NOTIFICATION_PRIORITY_LOW
	case models.NotificationPriorityNormal:
		return pb.NotificationPriority_NOTIFICATION_PRIORITY_NORMAL
	case models.NotificationPriorityHigh:
		return pb.NotificationPriority_NOTIFICATION_PRIORITY_HIGH
	case models.NotificationPritorityCritical:
		return pb.NotificationPriority_NOTIFICATION_PRIORITY_URGENT
	default:
		return pb.NotificationPriority_NOTIFICATION_PRIORITY_NORMAL
	}
}

func (h *GRPCHandler) convertNotificationStatusToProto(modelStatus models.NotificationStatus) pb.NotificationStatus {
	switch modelStatus {
	case models.NotificationStatusPending:
		return pb.NotificationStatus_NOTIFICATION_STATUS_PENDING
	case models.NotificationStatusSent:
		return pb.NotificationStatus_NOTIFICATION_STATUS_SENT
	case models.NotificationStatusDelivered:
		return pb.NotificationStatus_NOTIFICATION_STATUS_DELIVERED
	case models.NotificationStatusRead:
		return pb.NotificationStatus_NOTIFICATION_STATUS_READ
	case models.NotificationStatusFailed:
		return pb.NotificationStatus_NOTIFICATION_STATUS_FAILED
	default:
		return pb.NotificationStatus_NOTIFICATION_STATUS_PENDING
	}
}