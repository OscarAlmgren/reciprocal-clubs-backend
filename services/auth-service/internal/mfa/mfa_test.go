package mfa

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMFAService_GenerateSecret(t *testing.T) {
	service := NewMFAService("Test Issuer")

	secret, err := service.GenerateSecret("test@example.com")
	require.NoError(t, err)
	assert.NotEmpty(t, secret)
	assert.Greater(t, len(secret), 32) // Base32 encoded secret should be longer than raw bytes
}

func TestMFAService_GenerateQRCodeURL(t *testing.T) {
	service := NewMFAService("Test Issuer")
	secret := "JBSWY3DPEHPK3PXP"
	accountName := "test@example.com"

	url := service.GenerateQRCodeURL(accountName, secret)
	expected := "otpauth://totp/Test Issuer:test@example.com?secret=JBSWY3DPEHPK3PXP&issuer=Test Issuer"
	assert.Equal(t, expected, url)
}

func TestMFAService_VerifyTOTP(t *testing.T) {
	service := NewMFAService("Test Issuer")
	secret := "JBSWY3DPEHPK3PXP"

	// Generate current TOTP code
	currentCode, err := service.GetTOTPCurrentCode(secret)
	require.NoError(t, err)

	// Verify the current code
	valid := service.VerifyTOTP(secret, currentCode)
	assert.True(t, valid)

	// Test invalid code
	valid = service.VerifyTOTP(secret, "000000")
	assert.False(t, valid)
}

func TestMFAService_VerifyTOTPWithSkew(t *testing.T) {
	service := NewMFAService("Test Issuer")
	secret := "JBSWY3DPEHPK3PXP"

	// Generate current TOTP code
	currentCode, err := service.GetTOTPCurrentCode(secret)
	require.NoError(t, err)

	// Verify with skew tolerance
	valid := service.VerifyTOTPWithSkew(secret, currentCode, 1)
	assert.True(t, valid)

	// Test invalid code with skew
	valid = service.VerifyTOTPWithSkew(secret, "000000", 1)
	assert.False(t, valid)
}

func TestMFAService_GenerateBackupCodes(t *testing.T) {
	service := NewMFAService("Test Issuer")

	codes, err := service.GenerateBackupCodes(8)
	require.NoError(t, err)
	assert.Len(t, codes, 8)

	// Check format: XXXX-XXXX
	for _, code := range codes {
		assert.Len(t, code, 9)
		assert.Equal(t, "-", code[4:5])
	}

	// Ensure codes are unique
	codeMap := make(map[string]bool)
	for _, code := range codes {
		assert.False(t, codeMap[code], "Duplicate backup code: %s", code)
		codeMap[code] = true
	}
}

func TestMFAService_VerifyBackupCode(t *testing.T) {
	service := NewMFAService("Test Issuer")
	code := "ABCD-EFGH"
	storedCode := service.HashBackupCode(code)

	// Test valid backup code
	valid := service.VerifyBackupCode(code, storedCode)
	assert.True(t, valid)

	// Test invalid backup code
	valid = service.VerifyBackupCode("WRONG-CODE", storedCode)
	assert.False(t, valid)
}

func TestMFAService_GenerateSMSCode(t *testing.T) {
	service := NewMFAService("Test Issuer")

	code, err := service.GenerateSMSCode()
	require.NoError(t, err)
	assert.Len(t, code, 6)

	// Ensure it's numeric
	for _, char := range code {
		assert.True(t, char >= '0' && char <= '9', "Non-numeric character in SMS code: %c", char)
	}
}

func TestMFAService_GenerateEmailCode(t *testing.T) {
	service := NewMFAService("Test Issuer")

	code, err := service.GenerateEmailCode()
	require.NoError(t, err)
	assert.Len(t, code, 8)

	// Ensure it's numeric
	for _, char := range code {
		assert.True(t, char >= '0' && char <= '9', "Non-numeric character in email code: %c", char)
	}
}

func TestMFAService_GenerateRandomToken(t *testing.T) {
	service := NewMFAService("Test Issuer")

	token, err := service.GenerateRandomToken(32)
	require.NoError(t, err)
	assert.Len(t, token, 32)
}

func TestMFAService_ValidateCodeFormat(t *testing.T) {
	service := NewMFAService("Test Issuer")

	tests := []struct {
		code     string
		codeType string
		expected bool
	}{
		{"123456", "totp", true},
		{"12345", "totp", false},
		{"1234567", "totp", false},
		{"abcdef", "totp", false},
		{"ABCD-EFGH", "backup", true},
		{"ABCD_EFGH", "backup", false},
		{"ABCDEFGH", "backup", false},
		{"123456", "sms", true},
		{"12345", "sms", false},
		{"12345678", "email", true},
		{"1234567", "email", false},
	}

	for _, test := range tests {
		result := service.ValidateCodeFormat(test.code, test.codeType)
		assert.Equal(t, test.expected, result,
			"Code: %s, Type: %s", test.code, test.codeType)
	}
}

func TestMFAService_Integration(t *testing.T) {
	service := NewMFAService("Integration Test")
	accountName := "test@example.com"

	// Generate secret
	secret, err := service.GenerateSecret(accountName)
	require.NoError(t, err)

	// Generate QR code URL
	qrURL := service.GenerateQRCodeURL(accountName, secret)
	assert.Contains(t, qrURL, secret)
	assert.Contains(t, qrURL, accountName)

	// Generate backup codes
	backupCodes, err := service.GenerateBackupCodes(10)
	require.NoError(t, err)
	assert.Len(t, backupCodes, 10)

	// Test TOTP verification
	currentCode, err := service.GetTOTPCurrentCode(secret)
	require.NoError(t, err)

	valid := service.VerifyTOTP(secret, currentCode)
	assert.True(t, valid)

	// Test backup code verification
	backupCode := backupCodes[0]
	hashedCode := service.HashBackupCode(backupCode)

	valid = service.VerifyBackupCode(backupCode, hashedCode)
	assert.True(t, valid)
}

func BenchmarkMFAService_GenerateSecret(b *testing.B) {
	service := NewMFAService("Benchmark")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.GenerateSecret("test@example.com")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMFAService_VerifyTOTP(b *testing.B) {
	service := NewMFAService("Benchmark")
	secret := "JBSWY3DPEHPK3PXP"
	code, _ := service.GetTOTPCurrentCode(secret)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.VerifyTOTP(secret, code)
	}
}

func BenchmarkMFAService_GenerateBackupCodes(b *testing.B) {
	service := NewMFAService("Benchmark")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.GenerateBackupCodes(8)
		if err != nil {
			b.Fatal(err)
		}
	}
}