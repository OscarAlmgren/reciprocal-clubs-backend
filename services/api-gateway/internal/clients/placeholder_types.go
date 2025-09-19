package clients

import (
	"google.golang.org/protobuf/types/known/emptypb"
)

// Placeholder types for service requests/responses
// These will be replaced with actual protobuf types when services are integrated

// Auth Service Types
type RegisterUserRequest struct {
	ClubID   uint32
	Email    string
	Username string
}

type RegisterUserResponse struct {
	UserID  uint32
	Success bool
	Message string
}

type InitiatePasskeyLoginRequest struct {
	Email string
}

type InitiatePasskeyLoginResponse struct {
	Challenge []byte
	Success   bool
}

type CompletePasskeyLoginRequest struct {
	Email     string
	Challenge []byte
	Response  []byte
}

type CompletePasskeyLoginResponse struct {
	Token        string
	RefreshToken string
	UserID       uint32
	Success      bool
}

type ValidateSessionRequest struct {
	Token string
}

type ValidateSessionResponse struct {
	Valid  bool
	UserID uint32
	ClubID uint32
}

type LogoutRequest struct {
	UserID uint32
	Token  string
}

type LogoutResponse struct {
	Success bool
}

type GetUserWithRolesRequest struct {
	ClubID uint32
	UserID uint32
}

type GetUserWithRolesResponse struct {
	UserID      uint32
	Email       string
	Username    string
	Roles       []string
	Permissions []string
}

type UpdateUserRequest struct {
	ClubID   uint32
	UserID   uint32
	Email    string
	Username string
}

type UpdateUserResponse struct {
	Success bool
	Message string
}

type DeleteUserRequest struct {
	ClubID uint32
	UserID uint32
}

type DeleteUserResponse struct {
	Success bool
}

type CreateRoleRequest struct {
	ClubID      uint32
	Name        string
	Description string
	Permissions []string
}

type CreateRoleResponse struct {
	RoleID  uint32
	Success bool
}

type AssignRoleRequest struct {
	ClubID uint32
	UserID uint32
	RoleID uint32
}

type AssignRoleResponse struct {
	Success bool
}

type RemoveRoleRequest struct {
	ClubID uint32
	UserID uint32
	RoleID uint32
}

type RemoveRoleResponse struct {
	Success bool
}

type CheckPermissionRequest struct {
	ClubID     uint32
	UserID     uint32
	Permission string
}

type CheckPermissionResponse struct {
	Allowed bool
}

type GetUserPermissionsRequest struct {
	ClubID uint32
	UserID uint32
}

type GetUserPermissionsResponse struct {
	Permissions []string
}

type HealthCheckRequest = emptypb.Empty
type HealthCheckResponse struct {
	Status string
}

// Member Service Types
type CreateMemberRequest struct {
	ClubID         uint32
	UserID         uint32
	MembershipType string
}

type CreateMemberResponse struct {
	MemberID     uint32
	MemberNumber string
	Success      bool
}

type GetMemberRequest struct {
	ClubID   uint32
	MemberID uint32
}

type GetMemberResponse struct {
	MemberID       uint32
	MemberNumber   string
	MembershipType string
	Status         string
}

type UpdateMemberRequest struct {
	ClubID   uint32
	MemberID uint32
	Status   string
}

type UpdateMemberResponse struct {
	Success bool
}

type DeleteMemberRequest struct {
	ClubID   uint32
	MemberID uint32
}

type DeleteMemberResponse struct {
	Success bool
}

type ListMembersRequest struct {
	ClubID uint32
	Limit  int32
	Offset int32
}

type ListMembersResponse struct {
	Members []Member
	Total   int32
}

type Member struct {
	MemberID       uint32
	MemberNumber   string
	MembershipType string
	Status         string
}

type SearchMembersRequest struct {
	ClubID uint32
	Query  string
	Limit  int32
}

type SearchMembersResponse struct {
	Members []Member
	Total   int32
}

type SuspendMemberRequest struct {
	ClubID   uint32
	MemberID uint32
	Reason   string
}

type SuspendMemberResponse struct {
	Success bool
}

type ActivateMemberRequest struct {
	ClubID   uint32
	MemberID uint32
}

type ActivateMemberResponse struct {
	Success bool
}

type GetMemberAnalyticsRequest struct {
	ClubID uint32
}

type GetMemberAnalyticsResponse struct {
	TotalMembers   int32
	ActiveMembers  int32
	NewThisMonth   int32
	MembershipTypes map[string]int32
}

// Reciprocal Service Types
type CreateAgreementRequest struct {
	ClubID        uint32
	PartnerClubID uint32
	Terms         string
}

type CreateAgreementResponse struct {
	AgreementID uint32
	Success     bool
}

type GetAgreementRequest struct {
	ClubID      uint32
	AgreementID uint32
}

type GetAgreementResponse struct {
	AgreementID   uint32
	PartnerClubID uint32
	Status        string
	Terms         string
}

type UpdateAgreementRequest struct {
	ClubID      uint32
	AgreementID uint32
	Status      string
	Terms       string
}

type UpdateAgreementResponse struct {
	Success bool
}

type ListAgreementsRequest struct {
	ClubID uint32
	Status string
	Limit  int32
}

type ListAgreementsResponse struct {
	Agreements []Agreement
	Total      int32
}

type Agreement struct {
	AgreementID   uint32
	PartnerClubID uint32
	Status        string
	Terms         string
}

type RequestVisitRequest struct {
	ClubID     uint32
	MemberID   uint32
	TargetClub uint32
	VisitDate  string
}

type RequestVisitResponse struct {
	VisitID uint32
	Success bool
}

type ConfirmVisitRequest struct {
	ClubID  uint32
	VisitID uint32
}

type ConfirmVisitResponse struct {
	Success bool
}

type CheckInVisitRequest struct {
	ClubID  uint32
	VisitID uint32
}

type CheckInVisitResponse struct {
	Success     bool
	CheckInTime string
}

type CheckOutVisitRequest struct {
	ClubID  uint32
	VisitID uint32
}

type CheckOutVisitResponse struct {
	Success      bool
	CheckOutTime string
}

type ListVisitsRequest struct {
	ClubID   uint32
	MemberID uint32
	Status   string
	Limit    int32
}

type ListVisitsResponse struct {
	Visits []Visit
	Total  int32
}

type Visit struct {
	VisitID       uint32
	MemberID      uint32
	TargetClub    uint32
	Status        string
	CheckInTime   string
	CheckOutTime  string
}

type GetVisitAnalyticsRequest struct {
	ClubID uint32
}

type GetVisitAnalyticsResponse struct {
	TotalVisits     int32
	VisitsThisMonth int32
	PopularClubs    map[string]int32
}

// Blockchain Service Types
type SubmitTransactionRequest struct {
	ClubID      uint32
	Type        string
	Data        []byte
	MemberID    uint32
}

type SubmitTransactionResponse struct {
	TransactionID string
	Success       bool
}

type GetTransactionRequest struct {
	TransactionID string
}

type GetTransactionResponse struct {
	TransactionID string
	Type          string
	Data          []byte
	Status        string
	Timestamp     string
}

type ListTransactionsRequest struct {
	ClubID   uint32
	MemberID uint32
	Type     string
	Limit    int32
}

type ListTransactionsResponse struct {
	Transactions []Transaction
	Total        int32
}

type Transaction struct {
	TransactionID string
	Type          string
	Data          []byte
	Status        string
	Timestamp     string
}

type QueryLedgerRequest struct {
	ClubID uint32
	Query  string
}

type QueryLedgerResponse struct {
	Results []map[string]interface{}
}

type GetBlockchainStatusRequest struct {
	ClubID uint32
}

type GetBlockchainStatusResponse struct {
	Status      string
	BlockHeight int64
	NodeCount   int32
}