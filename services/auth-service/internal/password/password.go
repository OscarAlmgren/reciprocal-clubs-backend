package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// PasswordService handles password-related operations
type PasswordService struct {
	tokenTTL time.Duration
}

// NewPasswordService creates a new password service
func NewPasswordService(tokenTTL time.Duration) *PasswordService {
	if tokenTTL == 0 {
		tokenTTL = 1 * time.Hour // Default 1 hour
	}
	return &PasswordService{
		tokenTTL: tokenTTL,
	}
}

// HashPassword hashes a password using bcrypt
func (p *PasswordService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

// VerifyPassword verifies a password against its hash
func (p *PasswordService) VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateResetToken generates a secure password reset token
func (p *PasswordService) GenerateResetToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate reset token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GenerateResetTokenWithExpiry generates a reset token with expiry time
func (p *PasswordService) GenerateResetTokenWithExpiry() (string, time.Time, error) {
	token, err := p.GenerateResetToken()
	if err != nil {
		return "", time.Time{}, err
	}

	expiry := time.Now().Add(p.tokenTTL)
	return token, expiry, nil
}

// ValidateResetToken validates a password reset token
func (p *PasswordService) ValidateResetToken(providedToken, storedToken string, expiresAt *time.Time) bool {
	// Check if token exists and hasn't expired
	if storedToken == "" || expiresAt == nil {
		return false
	}

	if expiresAt.Before(time.Now()) {
		return false
	}

	// Use constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(providedToken), []byte(storedToken)) == 1
}

// ValidatePasswordStrength validates password strength
func (p *PasswordService) ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	if len(password) > 128 {
		return fmt.Errorf("password must be less than 128 characters long")
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		case isSpecialChar(char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}

	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}

	if !hasDigit {
		return fmt.Errorf("password must contain at least one digit")
	}

	if !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}

	return nil
}

// isSpecialChar checks if a character is a special character
func isSpecialChar(char rune) bool {
	specialChars := "!@#$%^&*()_+-=[]{}|;:,.<>?"
	for _, special := range specialChars {
		if char == special {
			return true
		}
	}
	return false
}

// GenerateTemporaryPassword generates a temporary password
func (p *PasswordService) GenerateTemporaryPassword(length int) (string, error) {
	if length < 8 {
		length = 12 // Default minimum length
	}

	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	bytes := make([]byte, length)

	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate temporary password: %w", err)
	}

	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}

	return string(bytes), nil
}

// IsPasswordCompromised checks if a password is in a common password list
// This is a simplified version - in production, you'd check against a real breach database
func (p *PasswordService) IsPasswordCompromised(password string) bool {
	commonPasswords := []string{
		"password", "123456", "123456789", "12345678", "12345",
		"password123", "admin", "qwerty", "abc123", "letmein",
		"monkey", "dragon", "111111", "123123", "football",
	}

	for _, common := range commonPasswords {
		if password == common {
			return true
		}
	}

	return false
}

// GetPasswordStrengthScore returns a password strength score (0-100)
func (p *PasswordService) GetPasswordStrengthScore(password string) int {
	score := 0

	// Length bonus
	if len(password) >= 8 {
		score += 25
	}
	if len(password) >= 12 {
		score += 10
	}
	if len(password) >= 16 {
		score += 5
	}

	// Character variety bonus
	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		case isSpecialChar(char):
			hasSpecial = true
		}
	}

	if hasUpper {
		score += 15
	}
	if hasLower {
		score += 15
	}
	if hasDigit {
		score += 15
	}
	if hasSpecial {
		score += 15
	}

	// Penalty for common passwords
	if p.IsPasswordCompromised(password) {
		score -= 50
	}

	// Ensure score is between 0 and 100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}