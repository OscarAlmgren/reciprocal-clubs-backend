package password

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasswordService_HashPassword(t *testing.T) {
	service := NewPasswordService(1 * time.Hour)
	password := "testPassword123!"

	hash, err := service.HashPassword(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)
}

func TestPasswordService_VerifyPassword(t *testing.T) {
	service := NewPasswordService(1 * time.Hour)
	password := "testPassword123!"

	hash, err := service.HashPassword(password)
	require.NoError(t, err)

	// Test correct password
	valid := service.VerifyPassword(password, hash)
	assert.True(t, valid)

	// Test incorrect password
	valid = service.VerifyPassword("wrongPassword", hash)
	assert.False(t, valid)
}

func TestPasswordService_GenerateResetToken(t *testing.T) {
	service := NewPasswordService(1 * time.Hour)

	token, err := service.GenerateResetToken()
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Greater(t, len(token), 40) // Base64 encoded 32 bytes should be > 40 chars
}

func TestPasswordService_GenerateResetTokenWithExpiry(t *testing.T) {
	service := NewPasswordService(2 * time.Hour)

	token, expiry, err := service.GenerateResetTokenWithExpiry()
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.True(t, expiry.After(time.Now()))
	assert.True(t, expiry.Before(time.Now().Add(3*time.Hour)))
}

func TestPasswordService_ValidateResetToken(t *testing.T) {
	service := NewPasswordService(1 * time.Hour)

	token, expiry, err := service.GenerateResetTokenWithExpiry()
	require.NoError(t, err)

	// Test valid token
	valid := service.ValidateResetToken(token, token, &expiry)
	assert.True(t, valid)

	// Test wrong token
	valid = service.ValidateResetToken("wrongtoken", token, &expiry)
	assert.False(t, valid)

	// Test expired token
	pastTime := time.Now().Add(-1 * time.Hour)
	valid = service.ValidateResetToken(token, token, &pastTime)
	assert.False(t, valid)

	// Test nil expiry
	valid = service.ValidateResetToken(token, token, nil)
	assert.False(t, valid)

	// Test empty stored token
	valid = service.ValidateResetToken(token, "", &expiry)
	assert.False(t, valid)
}

func TestPasswordService_ValidatePasswordStrength(t *testing.T) {
	service := NewPasswordService(1 * time.Hour)

	tests := []struct {
		password string
		valid    bool
		desc     string
	}{
		{"Password123!", true, "valid strong password"},
		{"pass", false, "too short"},
		{"password", false, "no uppercase"},
		{"PASSWORD", false, "no lowercase"},
		{"Password", false, "no digit"},
		{"Password123", false, "no special character"},
		{"Pass123!", true, "minimum requirements met"},
		{"", false, "empty password"},
		{string(make([]byte, 200)), false, "too long"},
		{"MyVeryLongAndComplexPassword123!@#", true, "long valid password"},
	}

	for _, test := range tests {
		err := service.ValidatePasswordStrength(test.password)
		if test.valid {
			assert.NoError(t, err, "Password should be valid: %s (%s)", test.password, test.desc)
		} else {
			assert.Error(t, err, "Password should be invalid: %s (%s)", test.password, test.desc)
		}
	}
}

func TestPasswordService_GenerateTemporaryPassword(t *testing.T) {
	service := NewPasswordService(1 * time.Hour)

	// Test default length
	password, err := service.GenerateTemporaryPassword(12)
	require.NoError(t, err)
	assert.Len(t, password, 12)

	// Test minimum length enforcement
	password, err = service.GenerateTemporaryPassword(4)
	require.NoError(t, err)
	assert.Len(t, password, 12) // Should default to 12

	// Test longer password
	password, err = service.GenerateTemporaryPassword(20)
	require.NoError(t, err)
	assert.Len(t, password, 20)

	// Verify it meets strength requirements
	err = service.ValidatePasswordStrength(password)
	assert.NoError(t, err)
}

func TestPasswordService_IsPasswordCompromised(t *testing.T) {
	service := NewPasswordService(1 * time.Hour)

	// Test common passwords
	compromised := service.IsPasswordCompromised("password")
	assert.True(t, compromised)

	compromised = service.IsPasswordCompromised("123456")
	assert.True(t, compromised)

	// Test unique password
	compromised = service.IsPasswordCompromised("UniquePassword123!")
	assert.False(t, compromised)
}

func TestPasswordService_GetPasswordStrengthScore(t *testing.T) {
	service := NewPasswordService(1 * time.Hour)

	tests := []struct {
		password    string
		minScore    int
		maxScore    int
		description string
	}{
		{"password", 0, 30, "weak common password"},
		{"Password123!", 80, 100, "strong password"},
		{"Pass1!", 60, 85, "decent password"},
		{"VeryLongAndComplexPassword123!@#$", 85, 100, "very strong password"},
		{"", 0, 20, "empty password"},
	}

	for _, test := range tests {
		score := service.GetPasswordStrengthScore(test.password)
		assert.GreaterOrEqual(t, score, test.minScore,
			"Score too low for '%s': got %d, expected >= %d (%s)",
			test.password, score, test.minScore, test.description)
		assert.LessOrEqual(t, score, test.maxScore,
			"Score too high for '%s': got %d, expected <= %d (%s)",
			test.password, score, test.maxScore, test.description)
	}
}

func TestPasswordService_DefaultTTL(t *testing.T) {
	// Test default TTL when 0 is provided
	service := NewPasswordService(0)

	token, expiry, err := service.GenerateResetTokenWithExpiry()
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Should default to 1 hour
	expectedExpiry := time.Now().Add(1 * time.Hour)
	assert.WithinDuration(t, expectedExpiry, expiry, 1*time.Minute)
}

func TestPasswordService_Integration(t *testing.T) {
	service := NewPasswordService(30 * time.Minute)
	originalPassword := "OriginalPassword123!"
	newPassword := "NewPassword456@"

	// Hash original password
	originalHash, err := service.HashPassword(originalPassword)
	require.NoError(t, err)

	// Verify original password
	valid := service.VerifyPassword(originalPassword, originalHash)
	assert.True(t, valid)

	// Generate reset token
	resetToken, expiry, err := service.GenerateResetTokenWithExpiry()
	require.NoError(t, err)

	// Validate reset token
	valid = service.ValidateResetToken(resetToken, resetToken, &expiry)
	assert.True(t, valid)

	// Validate new password strength
	err = service.ValidatePasswordStrength(newPassword)
	assert.NoError(t, err)

	// Check password is not compromised
	compromised := service.IsPasswordCompromised(newPassword)
	assert.False(t, compromised)

	// Hash new password
	newHash, err := service.HashPassword(newPassword)
	require.NoError(t, err)

	// Verify new password
	valid = service.VerifyPassword(newPassword, newHash)
	assert.True(t, valid)

	// Ensure old password doesn't work with new hash
	valid = service.VerifyPassword(originalPassword, newHash)
	assert.False(t, valid)
}

func BenchmarkPasswordService_HashPassword(b *testing.B) {
	service := NewPasswordService(1 * time.Hour)
	password := "BenchmarkPassword123!"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.HashPassword(password)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPasswordService_VerifyPassword(b *testing.B) {
	service := NewPasswordService(1 * time.Hour)
	password := "BenchmarkPassword123!"
	hash, _ := service.HashPassword(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.VerifyPassword(password, hash)
	}
}

func BenchmarkPasswordService_GenerateResetToken(b *testing.B) {
	service := NewPasswordService(1 * time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.GenerateResetToken()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPasswordService_ValidatePasswordStrength(b *testing.B) {
	service := NewPasswordService(1 * time.Hour)
	password := "BenchmarkPassword123!"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ValidatePasswordStrength(password)
	}
}