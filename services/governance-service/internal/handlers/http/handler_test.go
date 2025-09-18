package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"reciprocal-clubs-backend/services/governance-service/internal/models"
	"reciprocal-clubs-backend/services/governance-service/internal/service"
)

// Mock service for testing
type mockService struct {
	proposals     map[uint]*models.Proposal
	votes         map[uint]*models.Vote
	votingRights  map[string]*models.VotingRights
	policies      map[uint]*models.GovernancePolicy
	nextID        uint
	shouldError   bool
}

func newMockService() *mockService {
	return &mockService{
		proposals:    make(map[uint]*models.Proposal),
		votes:        make(map[uint]*models.Vote),
		votingRights: make(map[string]*models.VotingRights),
		policies:     make(map[uint]*models.GovernancePolicy),
		nextID:       1,
	}
}

func (m *mockService) CreateProposal(ctx context.Context, req *service.CreateProposalRequest) (*models.Proposal, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}

	proposal := &models.Proposal{
		ID:               m.nextID,
		ClubID:           req.ClubID,
		Title:            req.Title,
		Description:      req.Description,
		Type:             req.Type,
		Status:           models.ProposalStatusDraft,
		ProposerID:       req.ProposerID,
		VotingMethod:     req.VotingMethod,
		QuorumRequired:   req.QuorumRequired,
		MajorityRequired: req.MajorityRequired,
		VotingStartTime:  req.VotingStartTime,
		VotingEndTime:    req.VotingEndTime,
		Metadata:         req.Metadata,
		CreatedAt:        time.Now(),
	}

	m.proposals[proposal.ID] = proposal
	m.nextID++
	return proposal, nil
}

func (m *mockService) GetProposal(ctx context.Context, id uint) (*models.Proposal, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}

	proposal, exists := m.proposals[id]
	if !exists {
		return nil, fmt.Errorf("proposal not found")
	}
	return proposal, nil
}

func (m *mockService) GetProposalsByClub(ctx context.Context, clubID uint) ([]models.Proposal, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}

	var proposals []models.Proposal
	for _, proposal := range m.proposals {
		if proposal.ClubID == clubID {
			proposals = append(proposals, *proposal)
		}
	}
	return proposals, nil
}

func (m *mockService) GetActiveProposals(ctx context.Context, clubID uint) ([]models.Proposal, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}

	var proposals []models.Proposal
	for _, proposal := range m.proposals {
		if proposal.ClubID == clubID && proposal.Status == models.ProposalStatusActive {
			proposals = append(proposals, *proposal)
		}
	}
	return proposals, nil
}

func (m *mockService) ActivateProposal(ctx context.Context, proposalID, activatorID uint) (*models.Proposal, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}

	proposal, exists := m.proposals[proposalID]
	if !exists {
		return nil, fmt.Errorf("proposal not found")
	}

	proposal.Status = models.ProposalStatusActive
	return proposal, nil
}

func (m *mockService) FinalizeProposal(ctx context.Context, proposalID uint) (*models.Proposal, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}

	proposal, exists := m.proposals[proposalID]
	if !exists {
		return nil, fmt.Errorf("proposal not found")
	}

	proposal.Status = models.ProposalStatusPassed
	return proposal, nil
}

func (m *mockService) CastVote(ctx context.Context, req *service.CastVoteRequest) (*models.Vote, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}

	vote := &models.Vote{
		ID:         m.nextID,
		ProposalID: req.ProposalID,
		MemberID:   req.MemberID,
		Choice:     req.Choice,
		Reason:     req.Reason,
		Metadata:   req.Metadata,
		VotedAt:    time.Now(),
		CreatedAt:  time.Now(),
	}

	m.votes[vote.ID] = vote
	m.nextID++
	return vote, nil
}

func (m *mockService) CreateVotingRights(ctx context.Context, req *service.CreateVotingRightsRequest) (*models.VotingRights, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}

	rights := &models.VotingRights{
		ID:             m.nextID,
		MemberID:       req.MemberID,
		ClubID:         req.ClubID,
		CanVote:        req.CanVote,
		CanPropose:     req.CanPropose,
		VotingWeight:   req.VotingWeight,
		Role:           req.Role,
		Restrictions:   req.Restrictions,
		Metadata:       req.Metadata,
		EffectiveFrom:  req.EffectiveFrom,
		EffectiveUntil: req.EffectiveUntil,
		CreatedAt:      time.Now(),
	}

	key := fmt.Sprintf("%d:%d", req.MemberID, req.ClubID)
	m.votingRights[key] = rights
	m.nextID++
	return rights, nil
}

func (m *mockService) GetVotingRights(ctx context.Context, memberID, clubID uint) (*models.VotingRights, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}

	key := fmt.Sprintf("%d:%d", memberID, clubID)
	rights, exists := m.votingRights[key]
	if !exists {
		return nil, fmt.Errorf("voting rights not found")
	}
	return rights, nil
}

func (m *mockService) CreateGovernancePolicy(ctx context.Context, req *service.CreateGovernancePolicyRequest) (*models.GovernancePolicy, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}

	policy := &models.GovernancePolicy{
		ID:             m.nextID,
		ClubID:         req.ClubID,
		Name:           req.Name,
		Description:    req.Description,
		PolicyType:     req.PolicyType,
		Rules:          req.Rules,
		IsActive:       req.IsActive,
		EffectiveFrom:  req.EffectiveFrom,
		EffectiveUntil: req.EffectiveUntil,
		CreatedBy:      req.CreatedBy,
		Metadata:       req.Metadata,
		CreatedAt:      time.Now(),
	}

	m.policies[policy.ID] = policy
	m.nextID++
	return policy, nil
}

func (m *mockService) GetActiveGovernancePolicies(ctx context.Context, clubID uint) ([]models.GovernancePolicy, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}

	var policies []models.GovernancePolicy
	for _, policy := range m.policies {
		if policy.ClubID == clubID && policy.IsActive {
			policies = append(policies, *policy)
		}
	}
	return policies, nil
}

func (m *mockService) HealthCheck(ctx context.Context) error {
	if m.shouldError {
		return fmt.Errorf("service unhealthy")
	}
	return nil
}

type mockLogger struct{}

func (m *mockLogger) Info(msg string, fields map[string]interface{})  {}
func (m *mockLogger) Error(msg string, fields map[string]interface{}) {}
func (m *mockLogger) Debug(msg string, fields map[string]interface{}) {}
func (m *mockLogger) Warn(msg string, fields map[string]interface{})  {}
func (m *mockLogger) Fatal(msg string, fields map[string]interface{}) {}
func (m *mockLogger) With(fields map[string]interface{}) interface{} { return m }

type mockMonitoring struct{}

func (m *mockMonitoring) RecordBusinessEvent(event, value string) {}
func (m *mockMonitoring) RecordHTTPRequest(method, path string, status int, duration time.Duration) {}
func (m *mockMonitoring) StartMetricsServer() {}

func setupTestHandler(mockSvc *mockService) *HTTPHandler {
	logger := &mockLogger{}
	monitoring := &mockMonitoring{}

	return NewHTTPHandler(mockSvc, logger, monitoring)
}

func TestHTTPHandler_HealthCheck(t *testing.T) {
	mockSvc := newMockService()
	handler := setupTestHandler(mockSvc)

	tests := []struct {
		name         string
		shouldError  bool
		expectedCode int
	}{
		{
			name:         "Healthy service",
			shouldError:  false,
			expectedCode: http.StatusOK,
		},
		{
			name:         "Unhealthy service",
			shouldError:  true,
			expectedCode: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc.shouldError = tt.shouldError

			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()

			handler.healthCheck(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("healthCheck() status = %d, want %d", w.Code, tt.expectedCode)
			}
		})
	}
}

func TestHTTPHandler_CreateProposal(t *testing.T) {
	mockSvc := newMockService()
	handler := setupTestHandler(mockSvc)

	tests := []struct {
		name         string
		requestBody  interface{}
		shouldError  bool
		expectedCode int
	}{
		{
			name: "Valid proposal",
			requestBody: service.CreateProposalRequest{
				ClubID:           1,
				Title:            "Test Proposal",
				Description:      "This is a test proposal description",
				Type:             models.ProposalTypePolicyChange,
				ProposerID:       1,
				VotingMethod:     models.VotingMethodSimpleMajority,
				QuorumRequired:   50,
				MajorityRequired: 50,
			},
			shouldError:  false,
			expectedCode: http.StatusCreated,
		},
		{
			name:         "Invalid JSON",
			requestBody:  "invalid json",
			shouldError:  false,
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "Service error",
			requestBody: service.CreateProposalRequest{
				ClubID:      1,
				Title:       "Test Proposal",
				Description: "Description",
				ProposerID:  1,
			},
			shouldError:  true,
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc.shouldError = tt.shouldError

			var body []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req := httptest.NewRequest("POST", "/api/v1/proposals", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.createProposal(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("createProposal() status = %d, want %d", w.Code, tt.expectedCode)
			}

			if tt.expectedCode == http.StatusCreated {
				var response models.Proposal
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}

				expected := tt.requestBody.(service.CreateProposalRequest)
				if response.Title != expected.Title {
					t.Errorf("Response title = %s, want %s", response.Title, expected.Title)
				}
			}
		})
	}
}

func TestHTTPHandler_GetProposal(t *testing.T) {
	mockSvc := newMockService()
	handler := setupTestHandler(mockSvc)

	// Add a test proposal
	proposal := &models.Proposal{
		ID:          1,
		ClubID:      1,
		Title:       "Test Proposal",
		Description: "Description",
		Status:      models.ProposalStatusDraft,
		ProposerID:  1,
	}
	mockSvc.proposals[1] = proposal

	tests := []struct {
		name         string
		proposalID   string
		shouldError  bool
		expectedCode int
	}{
		{
			name:         "Valid proposal ID",
			proposalID:   "1",
			shouldError:  false,
			expectedCode: http.StatusOK,
		},
		{
			name:         "Invalid proposal ID",
			proposalID:   "invalid",
			shouldError:  false,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Non-existent proposal",
			proposalID:   "999",
			shouldError:  false,
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "Service error",
			proposalID:   "1",
			shouldError:  true,
			expectedCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc.shouldError = tt.shouldError

			req := httptest.NewRequest("GET", "/api/v1/proposals/"+tt.proposalID, nil)
			req = mux.SetURLVars(req, map[string]string{"id": tt.proposalID})
			w := httptest.NewRecorder()

			handler.getProposal(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("getProposal() status = %d, want %d", w.Code, tt.expectedCode)
			}

			if tt.expectedCode == http.StatusOK {
				var response models.Proposal
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}

				if response.Title != proposal.Title {
					t.Errorf("Response title = %s, want %s", response.Title, proposal.Title)
				}
			}
		})
	}
}

func TestHTTPHandler_ActivateProposal(t *testing.T) {
	mockSvc := newMockService()
	handler := setupTestHandler(mockSvc)

	// Add a test proposal
	proposal := &models.Proposal{
		ID:          1,
		ClubID:      1,
		Title:       "Test Proposal",
		Description: "Description",
		Status:      models.ProposalStatusDraft,
		ProposerID:  1,
	}
	mockSvc.proposals[1] = proposal

	tests := []struct {
		name         string
		proposalID   string
		requestBody  interface{}
		shouldError  bool
		expectedCode int
	}{
		{
			name:       "Valid activation",
			proposalID: "1",
			requestBody: map[string]interface{}{
				"activator_id": float64(1), // JSON numbers are float64
			},
			shouldError:  false,
			expectedCode: http.StatusOK,
		},
		{
			name:         "Invalid proposal ID",
			proposalID:   "invalid",
			requestBody:  map[string]interface{}{"activator_id": float64(1)},
			shouldError:  false,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Invalid JSON",
			proposalID:   "1",
			requestBody:  "invalid json",
			shouldError:  false,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Service error",
			proposalID:   "1",
			requestBody:  map[string]interface{}{"activator_id": float64(1)},
			shouldError:  true,
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc.shouldError = tt.shouldError

			var body []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req := httptest.NewRequest("POST", "/api/v1/proposals/"+tt.proposalID+"/activate", bytes.NewReader(body))
			req = mux.SetURLVars(req, map[string]string{"id": tt.proposalID})
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.activateProposal(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("activateProposal() status = %d, want %d", w.Code, tt.expectedCode)
			}

			if tt.expectedCode == http.StatusOK {
				var response models.Proposal
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}

				if response.Status != models.ProposalStatusActive {
					t.Errorf("Response status = %s, want %s", response.Status, models.ProposalStatusActive)
				}
			}
		})
	}
}

func TestHTTPHandler_CastVote(t *testing.T) {
	mockSvc := newMockService()
	handler := setupTestHandler(mockSvc)

	tests := []struct {
		name         string
		proposalID   string
		requestBody  interface{}
		shouldError  bool
		expectedCode int
	}{
		{
			name:       "Valid vote",
			proposalID: "1",
			requestBody: service.CastVoteRequest{
				MemberID: 1,
				Choice:   models.VoteChoiceYes,
				Reason:   "I support this proposal",
			},
			shouldError:  false,
			expectedCode: http.StatusCreated,
		},
		{
			name:         "Invalid proposal ID",
			proposalID:   "invalid",
			requestBody:  service.CastVoteRequest{MemberID: 1, Choice: models.VoteChoiceYes},
			shouldError:  false,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Invalid JSON",
			proposalID:   "1",
			requestBody:  "invalid json",
			shouldError:  false,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Service error",
			proposalID:   "1",
			requestBody:  service.CastVoteRequest{MemberID: 1, Choice: models.VoteChoiceYes},
			shouldError:  true,
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc.shouldError = tt.shouldError

			var body []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req := httptest.NewRequest("POST", "/api/v1/proposals/"+tt.proposalID+"/votes", bytes.NewReader(body))
			req = mux.SetURLVars(req, map[string]string{"id": tt.proposalID})
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.castVote(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("castVote() status = %d, want %d", w.Code, tt.expectedCode)
			}

			if tt.expectedCode == http.StatusCreated {
				var response models.Vote
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}

				expected := tt.requestBody.(service.CastVoteRequest)
				if response.Choice != expected.Choice {
					t.Errorf("Response choice = %s, want %s", response.Choice, expected.Choice)
				}
			}
		})
	}
}

func TestHTTPHandler_GetProposalsByClub(t *testing.T) {
	mockSvc := newMockService()
	handler := setupTestHandler(mockSvc)

	// Add test proposals
	proposals := []*models.Proposal{
		{ID: 1, ClubID: 1, Title: "Proposal 1", Status: models.ProposalStatusDraft},
		{ID: 2, ClubID: 1, Title: "Proposal 2", Status: models.ProposalStatusActive},
		{ID: 3, ClubID: 2, Title: "Proposal 3", Status: models.ProposalStatusDraft},
	}

	for _, proposal := range proposals {
		mockSvc.proposals[proposal.ID] = proposal
	}

	tests := []struct {
		name         string
		clubID       string
		shouldError  bool
		expectedCode int
		expectedCount int
	}{
		{
			name:          "Valid club ID",
			clubID:        "1",
			shouldError:   false,
			expectedCode:  http.StatusOK,
			expectedCount: 2,
		},
		{
			name:         "Invalid club ID",
			clubID:       "invalid",
			shouldError:  false,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Service error",
			clubID:       "1",
			shouldError:  true,
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc.shouldError = tt.shouldError

			req := httptest.NewRequest("GET", "/api/v1/clubs/"+tt.clubID+"/proposals", nil)
			req = mux.SetURLVars(req, map[string]string{"club_id": tt.clubID})
			w := httptest.NewRecorder()

			handler.getProposalsByClub(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("getProposalsByClub() status = %d, want %d", w.Code, tt.expectedCode)
			}

			if tt.expectedCode == http.StatusOK {
				var response []models.Proposal
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}

				if len(response) != tt.expectedCount {
					t.Errorf("Response count = %d, want %d", len(response), tt.expectedCount)
				}
			}
		})
	}
}

func TestHTTPHandler_CreateVotingRights(t *testing.T) {
	mockSvc := newMockService()
	handler := setupTestHandler(mockSvc)

	tests := []struct {
		name         string
		requestBody  interface{}
		shouldError  bool
		expectedCode int
	}{
		{
			name: "Valid voting rights",
			requestBody: service.CreateVotingRightsRequest{
				MemberID:      1,
				ClubID:        1,
				CanVote:       true,
				CanPropose:    false,
				VotingWeight:  1.0,
				Role:          "member",
				EffectiveFrom: time.Now(),
			},
			shouldError:  false,
			expectedCode: http.StatusCreated,
		},
		{
			name:         "Invalid JSON",
			requestBody:  "invalid json",
			shouldError:  false,
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "Service error",
			requestBody: service.CreateVotingRightsRequest{
				MemberID:      1,
				ClubID:        1,
				CanVote:       true,
				VotingWeight:  1.0,
				EffectiveFrom: time.Now(),
			},
			shouldError:  true,
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc.shouldError = tt.shouldError

			var body []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req := httptest.NewRequest("POST", "/api/v1/voting-rights", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.createVotingRights(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("createVotingRights() status = %d, want %d", w.Code, tt.expectedCode)
			}

			if tt.expectedCode == http.StatusCreated {
				var response models.VotingRights
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}

				expected := tt.requestBody.(service.CreateVotingRightsRequest)
				if response.MemberID != expected.MemberID {
					t.Errorf("Response member ID = %d, want %d", response.MemberID, expected.MemberID)
				}
			}
		})
	}
}

func TestHTTPHandler_GetVotingRights(t *testing.T) {
	mockSvc := newMockService()
	handler := setupTestHandler(mockSvc)

	// Add test voting rights
	rights := &models.VotingRights{
		ID:           1,
		MemberID:     1,
		ClubID:       1,
		CanVote:      true,
		VotingWeight: 1.0,
	}
	mockSvc.votingRights["1:1"] = rights

	tests := []struct {
		name         string
		memberID     string
		clubID       string
		shouldError  bool
		expectedCode int
	}{
		{
			name:         "Valid IDs",
			memberID:     "1",
			clubID:       "1",
			shouldError:  false,
			expectedCode: http.StatusOK,
		},
		{
			name:         "Invalid member ID",
			memberID:     "invalid",
			clubID:       "1",
			shouldError:  false,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Invalid club ID",
			memberID:     "1",
			clubID:       "invalid",
			shouldError:  false,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Service error",
			memberID:     "1",
			clubID:       "1",
			shouldError:  true,
			expectedCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc.shouldError = tt.shouldError

			req := httptest.NewRequest("GET", "/api/v1/members/"+tt.memberID+"/voting-rights/"+tt.clubID, nil)
			req = mux.SetURLVars(req, map[string]string{"member_id": tt.memberID, "club_id": tt.clubID})
			w := httptest.NewRecorder()

			handler.getVotingRights(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("getVotingRights() status = %d, want %d", w.Code, tt.expectedCode)
			}

			if tt.expectedCode == http.StatusOK {
				var response models.VotingRights
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}

				if response.MemberID != rights.MemberID {
					t.Errorf("Response member ID = %d, want %d", response.MemberID, rights.MemberID)
				}
			}
		})
	}
}

func TestHTTPHandler_Middleware(t *testing.T) {
	mockSvc := newMockService()
	handler := setupTestHandler(mockSvc)

	// Test that middleware doesn't break the request flow
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// Create a test handler to wrap with middleware
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Apply middleware
	wrappedHandler := handler.loggingMiddleware(handler.monitoringMiddleware(testHandler))

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Middleware test failed: status = %d, want %d", w.Code, http.StatusOK)
	}

	if w.Body.String() != "OK" {
		t.Errorf("Middleware test failed: body = %s, want %s", w.Body.String(), "OK")
	}
}
