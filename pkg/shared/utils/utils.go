package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

// StringPtr returns a pointer to the given string value
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to the given int value
func IntPtr(i int) *int {
	return &i
}

// UintPtr returns a pointer to the given uint value
func UintPtr(u uint) *uint {
	return &u
}

// BoolPtr returns a pointer to the given bool value
func BoolPtr(b bool) *bool {
	return &b
}

// TimePtr returns a pointer to the given time value
func TimePtr(t time.Time) *time.Time {
	return &t
}

// StringValue returns the value of the string pointer or empty string if nil
func StringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// IntValue returns the value of the int pointer or 0 if nil
func IntValue(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// UintValue returns the value of the uint pointer or 0 if nil
func UintValue(u *uint) uint {
	if u == nil {
		return 0
	}
	return *u
}

// BoolValue returns the value of the bool pointer or false if nil
func BoolValue(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// TimeValue returns the value of the time pointer or zero time if nil
func TimeValue(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

// GenerateUUID generates a new UUID string
func GenerateUUID() string {
	return uuid.New().String()
}

// GenerateRandomString generates a random string of given length
func GenerateRandomString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	bytes := make([]byte, length)
	
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	
	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}
	
	return string(bytes), nil
}

// GenerateRandomBytes generates random bytes of given length
func GenerateRandomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return bytes, nil
}

// HashSHA256 creates a SHA256 hash of the input string
func HashSHA256(input string) string {
	hasher := sha256.New()
	hasher.Write([]byte(input))
	return hex.EncodeToString(hasher.Sum(nil))
}

// HashBytes creates a SHA256 hash of the input bytes
func HashBytes(input []byte) string {
	hasher := sha256.New()
	hasher.Write(input)
	return hex.EncodeToString(hasher.Sum(nil))
}

// Validation functions

// IsValidEmail validates an email address
func IsValidEmail(email string) bool {
	if email == "" {
		return false
	}
	
	_, err := mail.ParseAddress(email)
	return err == nil
}

// IsValidUsername validates a username (alphanumeric, underscores, hyphens, 3-50 chars)
func IsValidUsername(username string) bool {
	if len(username) < 3 || len(username) > 50 {
		return false
	}
	
	matched, _ := regexp.MatchString("^[a-zA-Z0-9_-]+$", username)
	return matched
}

// IsValidPassword validates a password (at least 8 chars, mixed case, numbers)
func IsValidPassword(password string) bool {
	if len(password) < 8 {
		return false
	}
	
	var hasUpper, hasLower, hasNumber bool
	
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		}
	}
	
	return hasUpper && hasLower && hasNumber
}

// IsValidPhoneNumber validates a phone number (basic validation)
func IsValidPhoneNumber(phone string) bool {
	if phone == "" {
		return false
	}
	
	// Remove common formatting
	cleaned := strings.ReplaceAll(phone, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")
	cleaned = strings.ReplaceAll(cleaned, "+", "")
	
	// Check if it's all digits and reasonable length
	matched, _ := regexp.MatchString("^[0-9]{7,15}$", cleaned)
	return matched
}

// String manipulation functions

// TitleCase converts a string to title case
func TitleCase(s string) string {
	return strings.Title(strings.ToLower(s))
}

// SlugifyString converts a string to a URL-friendly slug
func SlugifyString(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)
	
	// Replace spaces and common punctuation with hyphens
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	
	// Remove leading/trailing hyphens
	s = strings.Trim(s, "-")
	
	return s
}

// TruncateString truncates a string to a maximum length with ellipsis
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	
	if maxLen <= 3 {
		return s[:maxLen]
	}
	
	return s[:maxLen-3] + "..."
}

// SanitizeString removes potentially harmful characters from a string
func SanitizeString(s string) string {
	// Remove null bytes and control characters
	re := regexp.MustCompile(`[\x00-\x1F\x7F]`)
	return re.ReplaceAllString(s, "")
}

// Array/Slice utility functions

// ContainsString checks if a string slice contains a specific string
func ContainsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ContainsUint checks if a uint slice contains a specific uint
func ContainsUint(slice []uint, item uint) bool {
	for _, u := range slice {
		if u == item {
			return true
		}
	}
	return false
}

// RemoveString removes all occurrences of a string from a slice
func RemoveString(slice []string, item string) []string {
	result := make([]string, 0)
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

// RemoveUint removes all occurrences of a uint from a slice
func RemoveUint(slice []uint, item uint) []uint {
	result := make([]uint, 0)
	for _, u := range slice {
		if u != item {
			result = append(result, u)
		}
	}
	return result
}

// UniqueStrings returns a slice with unique strings
func UniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	
	return result
}

// UniqueUints returns a slice with unique uints
func UniqueUints(slice []uint) []uint {
	seen := make(map[uint]bool)
	result := make([]uint, 0)
	
	for _, u := range slice {
		if !seen[u] {
			seen[u] = true
			result = append(result, u)
		}
	}
	
	return result
}

// Time utility functions

// BeginningOfDay returns the beginning of the day for the given time
func BeginningOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// EndOfDay returns the end of the day for the given time
func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}

// BeginningOfWeek returns the beginning of the week (Sunday) for the given time
func BeginningOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	return BeginningOfDay(t.AddDate(0, 0, -weekday))
}

// EndOfWeek returns the end of the week (Saturday) for the given time
func EndOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	return EndOfDay(t.AddDate(0, 0, 6-weekday))
}

// BeginningOfMonth returns the beginning of the month for the given time
func BeginningOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// EndOfMonth returns the end of the month for the given time
func EndOfMonth(t time.Time) time.Time {
	return BeginningOfMonth(t).AddDate(0, 1, 0).Add(-time.Nanosecond)
}

// IsBusinessDay checks if the given date is a business day (Monday-Friday)
func IsBusinessDay(t time.Time) bool {
	weekday := t.Weekday()
	return weekday >= time.Monday && weekday <= time.Friday
}

// AddBusinessDays adds business days to a date (skipping weekends)
func AddBusinessDays(t time.Time, days int) time.Time {
	result := t
	remainingDays := days
	
	for remainingDays > 0 {
		result = result.AddDate(0, 0, 1)
		if IsBusinessDay(result) {
			remainingDays--
		}
	}
	
	return result
}

// FormatDuration formats a duration in a human-readable format
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	
	if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
	
	days := int(d.Hours() / 24)
	return fmt.Sprintf("%dd", days)
}

// Pagination helpers

// PaginationParams holds pagination parameters
type PaginationParams struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Offset   int `json:"offset"`
	Total    int `json:"total"`
}

// NewPaginationParams creates pagination parameters with validation
func NewPaginationParams(page, pageSize int) *PaginationParams {
	if page < 1 {
		page = 1
	}
	
	if pageSize < 1 {
		pageSize = 10
	} else if pageSize > 100 {
		pageSize = 100
	}
	
	offset := (page - 1) * pageSize
	
	return &PaginationParams{
		Page:     page,
		PageSize: pageSize,
		Offset:   offset,
	}
}

// SetTotal sets the total count and returns the params for chaining
func (p *PaginationParams) SetTotal(total int) *PaginationParams {
	p.Total = total
	return p
}

// HasNextPage returns true if there are more pages
func (p *PaginationParams) HasNextPage() bool {
	return p.Offset+p.PageSize < p.Total
}

// HasPrevPage returns true if there are previous pages
func (p *PaginationParams) HasPrevPage() bool {
	return p.Page > 1
}

// TotalPages returns the total number of pages
func (p *PaginationParams) TotalPages() int {
	if p.Total == 0 || p.PageSize == 0 {
		return 0
	}
	
	return (p.Total + p.PageSize - 1) / p.PageSize
}

// Map utilities

// MapKeys returns all keys from a string map
func MapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// MapValues returns all values from a string map
func MapValues(m map[string]interface{}) []interface{} {
	values := make([]interface{}, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return values
}

// MergeMaps merges multiple maps into one (later maps override earlier ones)
func MergeMaps(maps ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	
	return result
}