package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"reciprocal-clubs-backend/services/reciprocal-service/internal/models"
	"reciprocal-clubs-backend/services/reciprocal-service/internal/service"
)

// Mock service interface
type MockService interface {
	CreateAgreement(ctx context.Context, req *service.CreateAgreementRequest) (*models.Agreement, error)
	GetAgreementByID(ctx context.Context, id uint) (*models.Agreement, error)
	GetAgreementsByClub(ctx context.Context, clubID uint) ([]models.Agreement, error)
	UpdateAgreementStatus(ctx context.Context, id uint, newStatus string, reviewedByID string) (*models.Agreement, error)
	RequestVisit(ctx context.Context, req *service.RequestVisitRequest) (*models.Visit, error)
	GetVisitByID(ctx context.Context, id uint) (*models.Visit, error)
	ConfirmVisit(ctx context.Context, id uint, confirmedByID string) (*models.Visit, error)
	CheckInVisit(ctx context.Context, verificationCode string) (*models.Visit, error)
	CheckOutVisit(ctx context.Context, verificationCode string, actualCost *float64) (*models.Visit, error)
	GetMemberVisits(ctx context.Context, memberID uint, limit, offset int) ([]models.Visit, error)
	GetClubVisits(ctx context.Context, clubID uint, limit, offset int) ([]models.Visit, error)
	GetMemberVisitStats(ctx context.Context, memberID uint, clubID uint, year int, month int) (*models.VisitStats, error)
}

// Mock service for testing
type mockService struct {
	agreements map[uint]*models.Agreement
	visits     map[uint]*models.Visit
	nextID     uint
	shouldError bool
	errorMessage string
}

func newMockService() *mockService {
	return &mockService{
		agreements: make(map[uint]*models.Agreement),
		visits:     make(map[uint]*models.Visit),
		nextID:     1,
	}
}

func (m *mockService) setError(message string) {
	m.shouldError = true
	m.errorMessage = message
}

func (m *mockService) clearError() {
	m.shouldError = false
	m.errorMessage = ""
}

func (m *mockService) CreateAgreement(ctx context.Context, req *service.CreateAgreementRequest) (*models.Agreement, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}
	agreement := &models.Agreement{
		ID:              m.nextID,
		ProposingClubID: req.ProposingClubID,
		TargetClubID:    req.TargetClubID,
		Title:           req.Title,
		Description:     req.Description,
		Terms:           req.Terms,
		Status:          models.AgreementStatusPending,
		ProposedAt:      time.Now(),
		ProposedByID:    req.ProposedByID,
	}
	m.nextID++
	m.agreements[agreement.ID] = agreement
	return agreement, nil
}

func (m *mockService) GetAgreementByID(ctx context.Context, id uint) (*models.Agreement, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}
	agreement, exists := m.agreements[id]
	if !exists {
		return nil, errors.New("agreement not found")
	}
	return agreement, nil
}

func (m *mockService) GetAgreementsByClub(ctx context.Context, clubID uint) ([]models.Agreement, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}
	var result []models.Agreement
	for _, agreement := range m.agreements {
		if agreement.ProposingClubID == clubID || agreement.TargetClubID == clubID {
			result = append(result, *agreement)
		}
	}
	return result, nil
}

func (m *mockService) UpdateAgreementStatus(ctx context.Context, id uint, newStatus string, reviewedByID string) (*models.Agreement, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}
	agreement, exists := m.agreements[id]
	if !exists {
		return nil, errors.New("agreement not found")
	}
	agreement.Status = models.AgreementStatus(newStatus)
	agreement.ReviewedByID = &reviewedByID
	now := time.Now()
	agreement.ReviewedAt = &now
	return agreement, nil
}

func (m *mockService) RequestVisit(ctx context.Context, req *service.RequestVisitRequest) (*models.Visit, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}
	visit := &models.Visit{
		ID:               m.nextID,
		AgreementID:      req.AgreementID,
		MemberID:         req.MemberID,
		VisitingClubID:   req.VisitingClubID,
		HomeClubID:       req.HomeClubID,
		VisitDate:        req.VisitDate,
		Purpose:          req.Purpose,
		GuestCount:       req.GuestCount,
		Status:           models.VisitStatusPending,
		VerificationCode: "test-code-" + string(rune(m.nextID)),
		EstimatedCost:    req.EstimatedCost,
		Currency:         req.Currency,
	}
	m.nextID++
	m.visits[visit.ID] = visit
	return visit, nil
}

func (m *mockService) GetVisitByID(ctx context.Context, id uint) (*models.Visit, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}
	visit, exists := m.visits[id]
	if !exists {
		return nil, errors.New("visit not found")
	}
	return visit, nil
}

func (m *mockService) ConfirmVisit(ctx context.Context, id uint, confirmedByID string) (*models.Visit, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}
	visit, exists := m.visits[id]
	if !exists {
		return nil, errors.New("visit not found")
	}
	visit.Status = models.VisitStatusConfirmed
	visit.VerifiedBy = &confirmedByID
	now := time.Now()
	visit.VerifiedAt = &now
	return visit, nil
}

func (m *mockService) CheckInVisit(ctx context.Context, verificationCode string) (*models.Visit, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}
	for _, visit := range m.visits {
		if visit.VerificationCode == verificationCode {
			visit.Status = models.VisitStatusCheckedIn
			now := time.Now()
			visit.CheckInTime = &now
			return visit, nil
		}
	}
	return nil, errors.New("visit not found")
}

func (m *mockService) CheckOutVisit(ctx context.Context, verificationCode string, actualCost *float64) (*models.Visit, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}
	for _, visit := range m.visits {
		if visit.VerificationCode == verificationCode {
			visit.Status = models.VisitStatusCompleted
			now := time.Now()
			visit.CheckOutTime = &now
			visit.ActualCost = actualCost
			return visit, nil
		}
	}
	return nil, errors.New("visit not found")
}

func (m *mockService) GetMemberVisits(ctx context.Context, memberID uint, limit, offset int) ([]models.Visit, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}
	var result []models.Visit
	for _, visit := range m.visits {
		if visit.MemberID == memberID {
			result = append(result, *visit)
		}
	}
	return result, nil
}

func (m *mockService) GetClubVisits(ctx context.Context, clubID uint, limit, offset int) ([]models.Visit, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}
	var result []models.Visit
	for _, visit := range m.visits {
		if visit.VisitingClubID == clubID {
			result = append(result, *visit)
		}
	}
	return result, nil
}

func (m *mockService) GetMemberVisitStats(ctx context.Context, memberID uint, clubID uint, year int, month int) (*models.VisitStats, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}
	return &models.VisitStats{
		MemberID:      memberID,
		ClubID:        clubID,
		Year:          year,
		Month:         month,
		VisitCount:    5,
		TotalDuration: 300,
		TotalCost:     150.0,
		AverageRating: 4.5,
	}, nil
}

// Mock logger
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, fields map[string]interface{}) {}
func (m *mockLogger) Info(msg string, fields map[string]interface{})  {}
func (m *mockLogger) Warn(msg string, fields map[string]interface{})  {}
func (m *mockLogger) Error(msg string, fields map[string]interface{}) {}
func (m *mockLogger) Fatal(msg string, fields map[string]interface{}) {}
func (m *mockLogger) With(fields map[string]interface{}) interface{} { return m }

// Mock monitoring
type mockMonitoring struct{}

func (m *mockMonitoring) RecordBusinessEvent(event, value string) {}
func (m *mockMonitoring) RecordHTTPRequest(method, path string, statusCode int, duration time.Duration) {}

// Test helper to create handler with mocks
func createTestHandler() (*HTTPHandler, *mockService) {
	service := newMockService()
	logger := &mockLogger{}
	monitoring := &mockMonitoring{}

	handler := NewHTTPHandler(service, logger, monitoring)
	return handler, service
}

func TestHTTPHandler_healthCheck(t *testing.T) {
	handler, _ := createTestHandler()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.healthCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("healthCheck() status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]string
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("healthCheck() status = %v, want %v", response["status"], "healthy")
	}
}

func TestHTTPHandler_createAgreement(t *testing.T) {
	handler, service := createTestHandler()

	t.Run("successful creation", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"proposing_club_id": 1,
			"target_club_id":    2,
			"title":             "Test Agreement",
			"description":       "Test Description",
			"terms": map[string]interface{}{
				"max_visits_per_month": 5,
				"max_visits_per_year":  60,
			},
			"proposed_by_id": "user123",
		}

		body, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/v1/agreements", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.createAgreement(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("createAgreement() status = %v, want %v", w.Code, http.StatusCreated)
		}

		var response models.Agreement
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}

		if response.Title != requestBody["title"] {
			t.Errorf("createAgreement() title = %v, want %v", response.Title, requestBody["title"])
		}
	})

	t.Run("invalid request body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/agreements", bytes.NewReader([]byte("invalid json")))
		w := httptest.NewRecorder()

		handler.createAgreement(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("createAgreement() status = %v, want %v", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("service error", func(t *testing.T) {
		service.setError("database error")
		defer service.clearError()

		requestBody := map[string]interface{}{
			"proposing_club_id": 1,
			"target_club_id":    2,
			"title":             "Test Agreement",
			"proposed_by_id":    "user123",
		}

		body, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/v1/agreements", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.createAgreement(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("createAgreement() status = %v, want %v", w.Code, http.StatusInternalServerError)
		}
	})
}

func TestHTTPHandler_getAgreement(t *testing.T) {
	handler, service := createTestHandler()

	// Create test agreement
	agreement := &models.Agreement{
		ID:              1,
		ProposingClubID: 1,
		TargetClubID:    2,
		Title:           "Test Agreement",
		Status:          models.AgreementStatusPending,
		ProposedAt:      time.Now(),
		ProposedByID:    "user123",
	}
	service.agreements[1] = agreement

	t.Run("existing agreement", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/agreements/1", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "1"})
		w := httptest.NewRecorder()

		handler.getAgreement(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("getAgreement() status = %v, want %v", w.Code, http.StatusOK)
		}

		var response models.Agreement
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}

		if response.ID != agreement.ID {
			t.Errorf("getAgreement() ID = %v, want %v", response.ID, agreement.ID)
		}
	})

	t.Run("invalid ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/agreements/invalid", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
		w := httptest.NewRecorder()

		handler.getAgreement(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("getAgreement() status = %v, want %v", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("non-existing agreement", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/agreements/999", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "999"})
		w := httptest.NewRecorder()

		handler.getAgreement(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("getAgreement() status = %v, want %v", w.Code, http.StatusNotFound)
		}
	})
}

func TestHTTPHandler_requestVisit(t *testing.T) {
	handler, _ := createTestHandler()

	t.Run("successful visit request", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"agreement_id":     1,
			"member_id":        123,
			"visiting_club_id": 2,
			"home_club_id":     1,
			"visit_date":       time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			"purpose":          "Business meeting",
			"guest_count":      2,
			"estimated_cost":   100.0,
			"currency":         "USD",
		}

		body, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/v1/visits", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.requestVisit(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("requestVisit() status = %v, want %v", w.Code, http.StatusCreated)
		}

		var response models.Visit
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}

		expectedMemberID := uint(requestBody["member_id"].(int))
		if response.MemberID != expectedMemberID {
			t.Errorf("requestVisit() MemberID = %v, want %v", response.MemberID, expectedMemberID)
		}
	})

	t.Run("invalid request body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/visits", bytes.NewReader([]byte("invalid json")))
		w := httptest.NewRecorder()

		handler.requestVisit(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("requestVisit() status = %v, want %v", w.Code, http.StatusBadRequest)
		}
	})
}

func TestHTTPHandler_checkInVisit(t *testing.T) {
	handler, service := createTestHandler()

	// Create test visit
	visit := &models.Visit{
		ID:               1,
		Status:           models.VisitStatusConfirmed,
		VerificationCode: "test-code-123",
	}
	service.visits[1] = visit

	t.Run("successful check in", func(t *testing.T) {
		requestBody := map[string]string{
			"verification_code": "test-code-123",
		}

		body, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/v1/visits/checkin", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.checkInVisit(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("checkInVisit() status = %v, want %v", w.Code, http.StatusOK)
		}

		var response models.Visit
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}

		if response.Status != models.VisitStatusCheckedIn {
			t.Errorf("checkInVisit() status = %v, want %v", response.Status, models.VisitStatusCheckedIn)
		}
	})

	t.Run("invalid verification code", func(t *testing.T) {
		requestBody := map[string]string{
			"verification_code": "invalid-code",
		}

		body, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/v1/visits/checkin", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.checkInVisit(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("checkInVisit() status = %v, want %v", w.Code, http.StatusInternalServerError)
		}
	})
}

func TestHTTPHandler_checkOutVisit(t *testing.T) {
	handler, service := createTestHandler()

	// Create test visit
	visit := &models.Visit{
		ID:               1,
		Status:           models.VisitStatusCheckedIn,
		VerificationCode: "test-code-123",
	}
	service.visits[1] = visit

	t.Run("successful check out with cost", func(t *testing.T) {
		actualCost := 150.0
		requestBody := map[string]interface{}{
			"verification_code": "test-code-123",
			"actual_cost":       actualCost,
		}

		body, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/v1/visits/checkout", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.checkOutVisit(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("checkOutVisit() status = %v, want %v", w.Code, http.StatusOK)
		}

		var response models.Visit
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}

		if response.Status != models.VisitStatusCompleted {
			t.Errorf("checkOutVisit() status = %v, want %v", response.Status, models.VisitStatusCompleted)
		}
	})

	t.Run("successful check out without cost", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"verification_code": "test-code-123",
		}

		body, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/v1/visits/checkout", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.checkOutVisit(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("checkOutVisit() status = %v, want %v", w.Code, http.StatusOK)
		}
	})
}

func TestHTTPHandler_getMemberStats(t *testing.T) {
	handler, _ := createTestHandler()

	t.Run("successful stats retrieval", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/members/123/stats?club_id=456&year=2024&month=3", nil)
		req = mux.SetURLVars(req, map[string]string{"memberId": "123"})
		w := httptest.NewRecorder()

		handler.getMemberStats(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("getMemberStats() status = %v, want %v", w.Code, http.StatusOK)
		}

		var response models.VisitStats
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}

		if response.MemberID != 123 {
			t.Errorf("getMemberStats() MemberID = %v, want %v", response.MemberID, 123)
		}
		if response.VisitCount != 5 {
			t.Errorf("getMemberStats() VisitCount = %v, want %v", response.VisitCount, 5)
		}
	})

	t.Run("invalid member ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/members/invalid/stats?club_id=456", nil)
		req = mux.SetURLVars(req, map[string]string{"memberId": "invalid"})
		w := httptest.NewRecorder()

		handler.getMemberStats(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("getMemberStats() status = %v, want %v", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("missing club ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/members/123/stats", nil)
		req = mux.SetURLVars(req, map[string]string{"memberId": "123"})
		w := httptest.NewRecorder()

		handler.getMemberStats(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("getMemberStats() status = %v, want %v", w.Code, http.StatusBadRequest)
		}
	})
}
