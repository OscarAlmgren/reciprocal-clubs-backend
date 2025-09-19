package clients

import "context"

// Enhanced service client interfaces with comprehensive methods

// AuthServiceClient provides authentication and authorization operations
type AuthServiceClient interface {
	Close() error
	HealthCheck(ctx context.Context) error

	// Authentication methods
	RegisterUser(ctx context.Context, req *RegisterUserRequest) (*RegisterUserResponse, error)
	InitiatePasskeyLogin(ctx context.Context, req *InitiatePasskeyLoginRequest) (*InitiatePasskeyLoginResponse, error)
	CompletePasskeyLogin(ctx context.Context, req *CompletePasskeyLoginRequest) (*CompletePasskeyLoginResponse, error)
	ValidateSession(ctx context.Context, req *ValidateSessionRequest) (*ValidateSessionResponse, error)
	Logout(ctx context.Context, req *LogoutRequest) (*LogoutResponse, error)

	// User management methods
	GetUserWithRoles(ctx context.Context, req *GetUserWithRolesRequest) (*GetUserWithRolesResponse, error)
	UpdateUser(ctx context.Context, req *UpdateUserRequest) (*UpdateUserResponse, error)
	DeleteUser(ctx context.Context, req *DeleteUserRequest) (*DeleteUserResponse, error)

	// Role management methods
	CreateRole(ctx context.Context, req *CreateRoleRequest) (*CreateRoleResponse, error)
	AssignRole(ctx context.Context, req *AssignRoleRequest) (*AssignRoleResponse, error)
	RemoveRole(ctx context.Context, req *RemoveRoleRequest) (*RemoveRoleResponse, error)

	// Permission methods
	CheckPermission(ctx context.Context, req *CheckPermissionRequest) (*CheckPermissionResponse, error)
	GetUserPermissions(ctx context.Context, req *GetUserPermissionsRequest) (*GetUserPermissionsResponse, error)
}

// MemberServiceClient provides member management operations
type MemberServiceClient interface {
	Close() error
	HealthCheck(ctx context.Context) error

	// Member CRUD operations
	CreateMember(ctx context.Context, req *CreateMemberRequest) (*CreateMemberResponse, error)
	GetMember(ctx context.Context, req *GetMemberRequest) (*GetMemberResponse, error)
	UpdateMember(ctx context.Context, req *UpdateMemberRequest) (*UpdateMemberResponse, error)
	DeleteMember(ctx context.Context, req *DeleteMemberRequest) (*DeleteMemberResponse, error)

	// Member listing and search
	ListMembers(ctx context.Context, req *ListMembersRequest) (*ListMembersResponse, error)
	SearchMembers(ctx context.Context, req *SearchMembersRequest) (*SearchMembersResponse, error)

	// Member status management
	SuspendMember(ctx context.Context, req *SuspendMemberRequest) (*SuspendMemberResponse, error)
	ActivateMember(ctx context.Context, req *ActivateMemberRequest) (*ActivateMemberResponse, error)

	// Member analytics
	GetMemberAnalytics(ctx context.Context, req *GetMemberAnalyticsRequest) (*GetMemberAnalyticsResponse, error)
}

// ReciprocalServiceClient provides reciprocal agreement and visit operations
type ReciprocalServiceClient interface {
	Close() error
	HealthCheck(ctx context.Context) error

	// Agreement operations
	CreateAgreement(ctx context.Context, req *CreateAgreementRequest) (*CreateAgreementResponse, error)
	GetAgreement(ctx context.Context, req *GetAgreementRequest) (*GetAgreementResponse, error)
	UpdateAgreement(ctx context.Context, req *UpdateAgreementRequest) (*UpdateAgreementResponse, error)
	ListAgreements(ctx context.Context, req *ListAgreementsRequest) (*ListAgreementsResponse, error)

	// Visit operations
	RequestVisit(ctx context.Context, req *RequestVisitRequest) (*RequestVisitResponse, error)
	ConfirmVisit(ctx context.Context, req *ConfirmVisitRequest) (*ConfirmVisitResponse, error)
	CheckInVisit(ctx context.Context, req *CheckInVisitRequest) (*CheckInVisitResponse, error)
	CheckOutVisit(ctx context.Context, req *CheckOutVisitRequest) (*CheckOutVisitResponse, error)

	// Visit management
	ListVisits(ctx context.Context, req *ListVisitsRequest) (*ListVisitsResponse, error)
	GetVisitAnalytics(ctx context.Context, req *GetVisitAnalyticsRequest) (*GetVisitAnalyticsResponse, error)
}

// BlockchainServiceClient provides blockchain transaction operations
type BlockchainServiceClient interface {
	Close() error
	HealthCheck(ctx context.Context) error

	// Transaction operations
	SubmitTransaction(ctx context.Context, req *SubmitTransactionRequest) (*SubmitTransactionResponse, error)
	GetTransaction(ctx context.Context, req *GetTransactionRequest) (*GetTransactionResponse, error)
	ListTransactions(ctx context.Context, req *ListTransactionsRequest) (*ListTransactionsResponse, error)

	// Blockchain queries
	QueryLedger(ctx context.Context, req *QueryLedgerRequest) (*QueryLedgerResponse, error)
	GetBlockchainStatus(ctx context.Context, req *GetBlockchainStatusRequest) (*GetBlockchainStatusResponse, error)
}

// Placeholder interfaces for remaining services (to be implemented when those services are completed)

type NotificationServiceClient interface {
	Close() error
	HealthCheck(ctx context.Context) error
	// Methods will be added when notification service is completed
}

type AnalyticsServiceClient interface {
	Close() error
	HealthCheck(ctx context.Context) error
	// Methods will be added when analytics service is completed
}

type GovernanceServiceClient interface {
	Close() error
	HealthCheck(ctx context.Context) error
	// Methods will be added when governance service is completed
}