package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"reciprocal-clubs-backend/pkg/shared/auth"
	"reciprocal-clubs-backend/services/api-gateway/internal/clients"

	"github.com/gorilla/mux"
)

// Request/Response types for HTTP API

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	ClubID   uint32 `json:"club_id"`
}

type LoginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
	ExpiresAt    string `json:"expires_at"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	ClubID   uint32 `json:"club_id"`
}

type RegisterResponse struct {
	UserID  uint32 `json:"user_id"`
	Message string `json:"message"`
	Success bool   `json:"success"`
}

type User struct {
	ID       uint32   `json:"id"`
	Email    string   `json:"email"`
	Username string   `json:"username"`
	ClubID   uint32   `json:"club_id"`
	Roles    []string `json:"roles"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// Authentication handlers

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Record authentication attempt
	s.gatewayMetrics.RecordAuthenticationAttempt("password", fmt.Sprintf("%d", req.ClubID), r.UserAgent())

	// Call auth service for login
	authReq := &clients.RegisterUserRequest{ // Using register request as placeholder for login
		ClubID:   req.ClubID,
		Email:    req.Email,
		Username: req.Email, // Placeholder
	}

	authResp, err := s.clients.AuthService.RegisterUser(r.Context(), authReq)
	if err != nil {
		s.gatewayMetrics.RecordAuthenticationFailure("password", fmt.Sprintf("%d", req.ClubID), "service_error")
		s.writeErrorResponse(w, http.StatusInternalServerError, "Authentication failed", err)
		return
	}

	s.gatewayMetrics.RecordAuthenticationSuccess("password", fmt.Sprintf("%d", req.ClubID), "member")

	response := LoginResponse{
		Token:        "mock-jwt-token",
		RefreshToken: "mock-refresh-token",
		User: User{
			ID:       authResp.UserID,
			Email:    req.Email,
			Username: req.Email,
			ClubID:   req.ClubID,
			Roles:    []string{"member"},
		},
		ExpiresAt: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	}

	s.writeJSONResponse(w, http.StatusOK, response)
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Call auth service
	authReq := &clients.RegisterUserRequest{
		ClubID:   req.ClubID,
		Email:    req.Email,
		Username: req.Username,
	}

	authResp, err := s.clients.AuthService.RegisterUser(r.Context(), authReq)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Registration failed", err)
		return
	}

	response := RegisterResponse{
		UserID:  authResp.UserID,
		Message: authResp.Message,
		Success: authResp.Success,
	}

	s.writeJSONResponse(w, http.StatusCreated, response)
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	// Placeholder implementation
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"token":         "new-mock-jwt-token",
		"refresh_token": "new-mock-refresh-token",
		"expires_at":    time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	})
}

func (s *Server) handleInitiatePasskey(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	email := req["email"]
	if email == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "Email is required", nil)
		return
	}

	passkeyReq := &clients.InitiatePasskeyLoginRequest{Email: email}
	passkeyResp, err := s.clients.AuthService.InitiatePasskeyLogin(r.Context(), passkeyReq)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Passkey initiation failed", err)
		return
	}

	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"challenge": passkeyResp.Challenge,
		"success":   passkeyResp.Success,
	})
}

func (s *Server) handleCompletePasskey(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Call auth service to complete passkey
	passkeyReq := &clients.CompletePasskeyLoginRequest{
		Email:     req["email"].(string),
		Challenge: []byte("mock-challenge"),
		Response:  []byte("mock-response"),
	}

	passkeyResp, err := s.clients.AuthService.CompletePasskeyLogin(r.Context(), passkeyReq)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Passkey completion failed", err)
		return
	}

	response := LoginResponse{
		Token:        passkeyResp.Token,
		RefreshToken: passkeyResp.RefreshToken,
		User: User{
			ID:       passkeyResp.UserID,
			Email:    req["email"].(string),
			Username: "passkeyuser",
			ClubID:   1,
			Roles:    []string{"member"},
		},
		ExpiresAt: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	}

	s.writeJSONResponse(w, http.StatusOK, response)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.writeErrorResponse(w, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	logoutReq := &clients.LogoutRequest{
		UserID: uint32(user.ID),
		Token:  "mock-token",
	}

	_, err := s.clients.AuthService.Logout(r.Context(), logoutReq)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Logout failed", err)
		return
	}

	s.writeJSONResponse(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.writeErrorResponse(w, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	userReq := &clients.GetUserWithRolesRequest{
		ClubID: uint32(user.ClubID),
		UserID: uint32(user.ID),
	}

	userResp, err := s.clients.AuthService.GetUserWithRoles(r.Context(), userReq)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get user", err)
		return
	}

	response := User{
		ID:       userResp.UserID,
		Email:    userResp.Email,
		Username: userResp.Username,
		ClubID:   uint32(user.ClubID),
		Roles:    userResp.Roles,
	}

	s.writeJSONResponse(w, http.StatusOK, response)
}

// User management handlers

func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)
	userID, _ := strconv.ParseUint(vars["userId"], 10, 32)

	userReq := &clients.GetUserWithRolesRequest{
		ClubID: uint32(clubID),
		UserID: uint32(userID),
	}

	userResp, err := s.clients.AuthService.GetUserWithRoles(r.Context(), userReq)
	if err != nil {
		s.writeErrorResponse(w, http.StatusNotFound, "User not found", err)
		return
	}

	response := User{
		ID:       userResp.UserID,
		Email:    userResp.Email,
		Username: userResp.Username,
		ClubID:   uint32(clubID),
		Roles:    userResp.Roles,
	}

	s.writeJSONResponse(w, http.StatusOK, response)
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)
	userID, _ := strconv.ParseUint(vars["userId"], 10, 32)

	var updateReq map[string]string
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	authReq := &clients.UpdateUserRequest{
		ClubID:   uint32(clubID),
		UserID:   uint32(userID),
		Email:    updateReq["email"],
		Username: updateReq["username"],
	}

	_, err := s.clients.AuthService.UpdateUser(r.Context(), authReq)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Update failed", err)
		return
	}

	s.writeJSONResponse(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)
	userID, _ := strconv.ParseUint(vars["userId"], 10, 32)

	deleteReq := &clients.DeleteUserRequest{
		ClubID: uint32(clubID),
		UserID: uint32(userID),
	}

	_, err := s.clients.AuthService.DeleteUser(r.Context(), deleteReq)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Deletion failed", err)
		return
	}

	s.writeJSONResponse(w, http.StatusOK, map[string]bool{"success": true})
}

// Member management handlers

func (s *Server) handleCreateMember(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	memberReq := &clients.CreateMemberRequest{
		ClubID:         uint32(req["club_id"].(float64)),
		UserID:         uint32(req["user_id"].(float64)),
		MembershipType: req["membership_type"].(string),
	}

	memberResp, err := s.clients.MemberService.CreateMember(r.Context(), memberReq)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Member creation failed", err)
		return
	}

	s.writeJSONResponse(w, http.StatusCreated, map[string]interface{}{
		"member_id":     memberResp.MemberID,
		"member_number": memberResp.MemberNumber,
		"success":       memberResp.Success,
	})
}

func (s *Server) handleListMembers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)

	// Parse query parameters
	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	listReq := &clients.ListMembersRequest{
		ClubID: uint32(clubID),
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	listResp, err := s.clients.MemberService.ListMembers(r.Context(), listReq)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to list members", err)
		return
	}

	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"members": listResp.Members,
		"total":   listResp.Total,
		"limit":   limit,
		"offset":  offset,
	})
}

func (s *Server) handleGetMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)
	memberID, _ := strconv.ParseUint(vars["memberId"], 10, 32)

	memberReq := &clients.GetMemberRequest{
		ClubID:   uint32(clubID),
		MemberID: uint32(memberID),
	}

	memberResp, err := s.clients.MemberService.GetMember(r.Context(), memberReq)
	if err != nil {
		s.writeErrorResponse(w, http.StatusNotFound, "Member not found", err)
		return
	}

	s.writeJSONResponse(w, http.StatusOK, memberResp)
}

func (s *Server) handleUpdateMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)
	memberID, _ := strconv.ParseUint(vars["memberId"], 10, 32)

	var updateReq map[string]string
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	memberReq := &clients.UpdateMemberRequest{
		ClubID:   uint32(clubID),
		MemberID: uint32(memberID),
		Status:   updateReq["status"],
	}

	_, err := s.clients.MemberService.UpdateMember(r.Context(), memberReq)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Update failed", err)
		return
	}

	s.writeJSONResponse(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) handleDeleteMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)
	memberID, _ := strconv.ParseUint(vars["memberId"], 10, 32)

	memberReq := &clients.DeleteMemberRequest{
		ClubID:   uint32(clubID),
		MemberID: uint32(memberID),
	}

	_, err := s.clients.MemberService.DeleteMember(r.Context(), memberReq)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Deletion failed", err)
		return
	}

	s.writeJSONResponse(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) handleSearchMembers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)
	query := r.URL.Query().Get("q")

	searchReq := &clients.SearchMembersRequest{
		ClubID: uint32(clubID),
		Query:  query,
		Limit:  50,
	}

	searchResp, err := s.clients.MemberService.SearchMembers(r.Context(), searchReq)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Search failed", err)
		return
	}

	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"members": searchResp.Members,
		"total":   searchResp.Total,
		"query":   query,
	})
}

func (s *Server) handleSuspendMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)
	memberID, _ := strconv.ParseUint(vars["memberId"], 10, 32)

	var req map[string]string
	json.NewDecoder(r.Body).Decode(&req)

	suspendReq := &clients.SuspendMemberRequest{
		ClubID:   uint32(clubID),
		MemberID: uint32(memberID),
		Reason:   req["reason"],
	}

	_, err := s.clients.MemberService.SuspendMember(r.Context(), suspendReq)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Suspension failed", err)
		return
	}

	s.writeJSONResponse(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) handleActivateMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)
	memberID, _ := strconv.ParseUint(vars["memberId"], 10, 32)

	activateReq := &clients.ActivateMemberRequest{
		ClubID:   uint32(clubID),
		MemberID: uint32(memberID),
	}

	_, err := s.clients.MemberService.ActivateMember(r.Context(), activateReq)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Activation failed", err)
		return
	}

	s.writeJSONResponse(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) handleMemberAnalytics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clubID, _ := strconv.ParseUint(vars["clubId"], 10, 32)

	analyticsReq := &clients.GetMemberAnalyticsRequest{
		ClubID: uint32(clubID),
	}

	analyticsResp, err := s.clients.MemberService.GetMemberAnalytics(r.Context(), analyticsReq)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Analytics failed", err)
		return
	}

	s.writeJSONResponse(w, http.StatusOK, analyticsResp)
}

// Placeholder implementations for remaining handlers
// These follow the same pattern: extract parameters, call service, return response

func (s *Server) handleCreateAgreement(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "Agreement creation endpoint - implementation in progress"})
}

func (s *Server) handleListAgreements(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "List agreements endpoint - implementation in progress"})
}

func (s *Server) handleGetAgreement(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "Get agreement endpoint - implementation in progress"})
}

func (s *Server) handleUpdateAgreement(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "Update agreement endpoint - implementation in progress"})
}

func (s *Server) handleRequestVisit(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "Request visit endpoint - implementation in progress"})
}

func (s *Server) handleConfirmVisit(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "Confirm visit endpoint - implementation in progress"})
}

func (s *Server) handleCheckInVisit(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "Check-in visit endpoint - implementation in progress"})
}

func (s *Server) handleCheckOutVisit(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "Check-out visit endpoint - implementation in progress"})
}

func (s *Server) handleListVisits(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "List visits endpoint - implementation in progress"})
}

func (s *Server) handleVisitAnalytics(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "Visit analytics endpoint - implementation in progress"})
}

func (s *Server) handleSubmitTransaction(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "Submit transaction endpoint - implementation in progress"})
}

func (s *Server) handleGetTransaction(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "Get transaction endpoint - implementation in progress"})
}

func (s *Server) handleListTransactions(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "List transactions endpoint - implementation in progress"})
}

func (s *Server) handleQueryLedger(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "Query ledger endpoint - implementation in progress"})
}

func (s *Server) handleBlockchainStatus(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "Blockchain status endpoint - implementation in progress"})
}

func (s *Server) handleCreateRole(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "Create role endpoint - implementation in progress"})
}

func (s *Server) handleAssignRole(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "Assign role endpoint - implementation in progress"})
}

func (s *Server) handleRemoveRole(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "Remove role endpoint - implementation in progress"})
}

func (s *Server) handleCheckPermission(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "Check permission endpoint - implementation in progress"})
}

func (s *Server) handleGetUserPermissions(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "Get user permissions endpoint - implementation in progress"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.writeErrorResponse(w, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	status := map[string]interface{}{
		"status":              "ok",
		"timestamp":           time.Now().Format(time.RFC3339),
		"service":             "api-gateway",
		"version":             s.config.Service.Version,
		"user_id":             user.ID,
		"club_id":             user.ClubID,
		"service_connections": s.clients.GetServiceStatus(r.Context()),
		"active_routes":       35,
		"uptime":              time.Since(time.Now()).String(), // Placeholder
	}

	s.writeJSONResponse(w, http.StatusOK, status)
}

func (s *Server) handleServiceConnections(w http.ResponseWriter, r *http.Request) {
	status := s.clients.GetServiceStatus(r.Context())
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"service_connections": status,
		"timestamp":           time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleRefreshConnections(w http.ResponseWriter, r *http.Request) {
	err := s.clients.RefreshConnections(s.config)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to refresh connections", err)
		return
	}
	s.writeJSONResponse(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) handleRateLimitStatus(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "Rate limit status endpoint - implementation in progress"})
}

func (s *Server) handleResetRateLimit(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "Reset rate limit endpoint - implementation in progress"})
}

func (s *Server) handleCircuitBreakerStatus(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "Circuit breaker status endpoint - implementation in progress"})
}

func (s *Server) handleResetCircuitBreaker(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "Reset circuit breaker endpoint - implementation in progress"})
}

func (s *Server) handleRequestAnalytics(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "Request analytics endpoint - implementation in progress"})
}

func (s *Server) handleGraphQLAnalytics(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"message": "GraphQL analytics endpoint - implementation in progress"})
}

// Helper methods

func (s *Server) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) writeErrorResponse(w http.ResponseWriter, statusCode int, message string, err error) {
	response := ErrorResponse{
		Error: message,
	}
	if err != nil {
		response.Details = err.Error()
		s.logger.Error("HTTP request error", map[string]interface{}{
			"status_code": statusCode,
			"message":     message,
			"error":       err.Error(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}