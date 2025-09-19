package mfa

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strconv"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// MFAService handles Multi-Factor Authentication operations
type MFAService struct {
	issuer string
}

// NewMFAService creates a new MFA service
func NewMFAService(issuer string) *MFAService {
	return &MFAService{
		issuer: issuer,
	}
}

// GenerateSecret generates a new TOTP secret for a user
func (m *MFAService) GenerateSecret(accountName string) (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      m.issuer,
		AccountName: accountName,
		SecretSize:  32,
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	return key.Secret(), nil
}

// GenerateQRCodeURL generates a QR code URL for TOTP setup
func (m *MFAService) GenerateQRCodeURL(accountName, secret string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s",
		m.issuer, accountName, secret, m.issuer)
}

// VerifyTOTP verifies a TOTP code against a secret
func (m *MFAService) VerifyTOTP(secret, code string) bool {
	return totp.Validate(code, secret)
}

// VerifyTOTPWithSkew verifies a TOTP code with time skew tolerance
func (m *MFAService) VerifyTOTPWithSkew(secret, code string, skew uint) bool {
	opts := totp.ValidateOpts{
		Period:    30,
		Skew:      skew,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	}
	valid, _ := totp.ValidateCustom(code, secret, time.Now(), opts)
	return valid
}

// GenerateBackupCodes generates backup codes for MFA
func (m *MFAService) GenerateBackupCodes(count int) ([]string, error) {
	codes := make([]string, count)

	for i := 0; i < count; i++ {
		code, err := m.generateBackupCode()
		if err != nil {
			return nil, fmt.Errorf("failed to generate backup code %d: %w", i, err)
		}
		codes[i] = code
	}

	return codes, nil
}

// generateBackupCode generates a single backup code
func (m *MFAService) generateBackupCode() (string, error) {
	// Generate 8 random bytes
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Convert to base32 and format nicely
	code := base32.StdEncoding.EncodeToString(bytes)

	// Format as XXXX-XXXX (remove padding and add hyphen)
	code = code[:8]
	return fmt.Sprintf("%s-%s", code[:4], code[4:]), nil
}

// HashBackupCode creates a hash of the backup code for storage
func (m *MFAService) HashBackupCode(code string) string {
	// In production, use proper hashing like bcrypt
	// For now, we'll store them directly (this should be changed)
	return code
}

// VerifyBackupCode verifies a backup code
func (m *MFAService) VerifyBackupCode(providedCode, storedCode string) bool {
	// In production, compare against hashed version
	return providedCode == storedCode
}

// GenerateSMSCode generates a 6-digit SMS verification code
func (m *MFAService) GenerateSMSCode() (string, error) {
	bytes := make([]byte, 3)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Convert to 6-digit code
	code := int(bytes[0])<<16 | int(bytes[1])<<8 | int(bytes[2])
	code = code % 1000000

	return fmt.Sprintf("%06d", code), nil
}

// GenerateEmailCode generates a 8-digit email verification code
func (m *MFAService) GenerateEmailCode() (string, error) {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Convert to 8-digit code
	code := int(bytes[0])<<24 | int(bytes[1])<<16 | int(bytes[2])<<8 | int(bytes[3])
	code = code % 100000000

	return fmt.Sprintf("%08d", code), nil
}

// GenerateRandomToken generates a random token for various purposes
func (m *MFAService) GenerateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return base32.StdEncoding.EncodeToString(bytes)[:length], nil
}

// ValidateCodeFormat validates the format of various codes
func (m *MFAService) ValidateCodeFormat(code, codeType string) bool {
	switch codeType {
	case "totp":
		// TOTP codes are 6 digits
		if len(code) != 6 {
			return false
		}
		_, err := strconv.Atoi(code)
		return err == nil
	case "backup":
		// Backup codes are in format XXXX-XXXX
		if len(code) != 9 {
			return false
		}
		return code[4] == '-'
	case "sms":
		// SMS codes are 6 digits
		if len(code) != 6 {
			return false
		}
		_, err := strconv.Atoi(code)
		return err == nil
	case "email":
		// Email codes are 8 digits
		if len(code) != 8 {
			return false
		}
		_, err := strconv.Atoi(code)
		return err == nil
	default:
		return false
	}
}

// GetTOTPCurrentCode gets the current TOTP code for testing purposes
func (m *MFAService) GetTOTPCurrentCode(secret string) (string, error) {
	return totp.GenerateCode(secret, time.Now())
}