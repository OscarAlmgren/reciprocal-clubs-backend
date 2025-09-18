package models

import (
	"testing"
	"time"
)

func TestAgreementStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status AgreementStatus
		want   bool
	}{
		{"Valid pending", AgreementStatusPending, true},
		{"Valid approved", AgreementStatusApproved, true},
		{"Valid rejected", AgreementStatusRejected, true},
		{"Valid active", AgreementStatusActive, true},
		{"Valid suspended", AgreementStatusSuspended, true},
		{"Valid expired", AgreementStatusExpired, true},
		{"Valid cancelled", AgreementStatusCancelled, true},
		{"Invalid status", AgreementStatus("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("AgreementStatus.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAgreement_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name        string
		currentStatus AgreementStatus
		newStatus     AgreementStatus
		want          bool
	}{
		// From pending
		{"Pending to approved", AgreementStatusPending, AgreementStatusApproved, true},
		{"Pending to rejected", AgreementStatusPending, AgreementStatusRejected, true},
		{"Pending to active", AgreementStatusPending, AgreementStatusActive, false},

		// From approved
		{"Approved to active", AgreementStatusApproved, AgreementStatusActive, true},
		{"Approved to cancelled", AgreementStatusApproved, AgreementStatusCancelled, true},
		{"Approved to pending", AgreementStatusApproved, AgreementStatusPending, false},

		// From active
		{"Active to suspended", AgreementStatusActive, AgreementStatusSuspended, true},
		{"Active to expired", AgreementStatusActive, AgreementStatusExpired, true},
		{"Active to cancelled", AgreementStatusActive, AgreementStatusCancelled, true},
		{"Active to approved", AgreementStatusActive, AgreementStatusApproved, false},

		// From suspended
		{"Suspended to active", AgreementStatusSuspended, AgreementStatusActive, true},
		{"Suspended to cancelled", AgreementStatusSuspended, AgreementStatusCancelled, true},
		{"Suspended to pending", AgreementStatusSuspended, AgreementStatusPending, false},

		// From terminal states
		{"Rejected to any", AgreementStatusRejected, AgreementStatusActive, false},
		{"Expired to any", AgreementStatusExpired, AgreementStatusActive, false},
		{"Cancelled to any", AgreementStatusCancelled, AgreementStatusActive, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agreement := &Agreement{Status: tt.currentStatus}
			if got := agreement.CanTransitionTo(tt.newStatus); got != tt.want {
				t.Errorf("Agreement.CanTransitionTo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAgreement_IsActive(t *testing.T) {
	tests := []struct {
		name   string
		status AgreementStatus
		want   bool
	}{
		{"Active status", AgreementStatusActive, true},
		{"Pending status", AgreementStatusPending, false},
		{"Suspended status", AgreementStatusSuspended, false},
		{"Expired status", AgreementStatusExpired, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agreement := &Agreement{Status: tt.status}
			if got := agreement.IsActive(); got != tt.want {
				t.Errorf("Agreement.IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAgreement_IsExpired(t *testing.T) {
	now := time.Now()
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{"No expiry date", nil, false},
		{"Expired", &past, true},
		{"Not expired", &future, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agreement := &Agreement{ExpiresAt: tt.expiresAt}
			if got := agreement.IsExpired(); got != tt.want {
				t.Errorf("Agreement.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVisitStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status VisitStatus
		want   bool
	}{
		{"Valid pending", VisitStatusPending, true},
		{"Valid confirmed", VisitStatusConfirmed, true},
		{"Valid checked in", VisitStatusCheckedIn, true},
		{"Valid completed", VisitStatusCompleted, true},
		{"Valid cancelled", VisitStatusCancelled, true},
		{"Valid no show", VisitStatusNoShow, true},
		{"Invalid status", VisitStatus("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("VisitStatus.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVisit_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name          string
		currentStatus VisitStatus
		newStatus     VisitStatus
		want          bool
	}{
		// From pending
		{"Pending to confirmed", VisitStatusPending, VisitStatusConfirmed, true},
		{"Pending to cancelled", VisitStatusPending, VisitStatusCancelled, true},
		{"Pending to checked in", VisitStatusPending, VisitStatusCheckedIn, false},

		// From confirmed
		{"Confirmed to checked in", VisitStatusConfirmed, VisitStatusCheckedIn, true},
		{"Confirmed to cancelled", VisitStatusConfirmed, VisitStatusCancelled, true},
		{"Confirmed to no show", VisitStatusConfirmed, VisitStatusNoShow, true},
		{"Confirmed to pending", VisitStatusConfirmed, VisitStatusPending, false},

		// From checked in
		{"Checked in to completed", VisitStatusCheckedIn, VisitStatusCompleted, true},
		{"Checked in to cancelled", VisitStatusCheckedIn, VisitStatusCancelled, false},

		// From terminal states
		{"Completed to any", VisitStatusCompleted, VisitStatusCheckedIn, false},
		{"Cancelled to any", VisitStatusCancelled, VisitStatusPending, false},
		{"No show to any", VisitStatusNoShow, VisitStatusConfirmed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visit := &Visit{Status: tt.currentStatus}
			if got := visit.CanTransitionTo(tt.newStatus); got != tt.want {
				t.Errorf("Visit.CanTransitionTo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVisit_IsCompleted(t *testing.T) {
	tests := []struct {
		name   string
		status VisitStatus
		want   bool
	}{
		{"Completed status", VisitStatusCompleted, true},
		{"Pending status", VisitStatusPending, false},
		{"Checked in status", VisitStatusCheckedIn, false},
		{"Cancelled status", VisitStatusCancelled, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visit := &Visit{Status: tt.status}
			if got := visit.IsCompleted(); got != tt.want {
				t.Errorf("Visit.IsCompleted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVisit_CalculateDuration(t *testing.T) {
	now := time.Now()
	checkIn := now.Add(-2 * time.Hour)
	checkOut := now

	tests := []struct {
		name         string
		checkInTime  *time.Time
		checkOutTime *time.Time
		want         *int
	}{
		{
			name:         "Both times set",
			checkInTime:  &checkIn,
			checkOutTime: &checkOut,
			want:         func() *int { d := 120; return &d }(), // 2 hours = 120 minutes
		},
		{
			name:         "Only check in time",
			checkInTime:  &checkIn,
			checkOutTime: nil,
			want:         nil,
		},
		{
			name:         "Only check out time",
			checkInTime:  nil,
			checkOutTime: &checkOut,
			want:         nil,
		},
		{
			name:         "No times set",
			checkInTime:  nil,
			checkOutTime: nil,
			want:         nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visit := &Visit{
				CheckInTime:  tt.checkInTime,
				CheckOutTime: tt.checkOutTime,
			}
			got := visit.CalculateDuration()
			if (got == nil && tt.want != nil) || (got != nil && tt.want == nil) {
				t.Errorf("Visit.CalculateDuration() = %v, want %v", got, tt.want)
				return
			}
			if got != nil && tt.want != nil && *got != *tt.want {
				t.Errorf("Visit.CalculateDuration() = %v, want %v", *got, *tt.want)
			}
		})
	}
}

func TestTableNames(t *testing.T) {
	tests := []struct {
		name  string
		model interface{ TableName() string }
		want  string
	}{
		{"Agreement table name", &Agreement{}, "reciprocal_agreements"},
		{"Visit table name", &Visit{}, "reciprocal_visits"},
		{"VisitRestriction table name", &VisitRestriction{}, "reciprocal_visit_restrictions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.model.TableName(); got != tt.want {
				t.Errorf("TableName() = %v, want %v", got, tt.want)
			}
		})
	}
}