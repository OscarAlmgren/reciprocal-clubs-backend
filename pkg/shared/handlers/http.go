package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"reciprocal-clubs-backend/pkg/shared/errors"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/utils"
)

// ResponseFormat represents different response formats
type ResponseFormat string

const (
	FormatJSON ResponseFormat = "json"
	FormatXML  ResponseFormat = "xml"
)

// StandardResponse represents a standard API response structure
type StandardResponse struct {
	Success   bool           `json:"success"`
	Data      interface{}    `json:"data,omitempty"`
	Error     *ErrorResponse `json:"error,omitempty"`
	Meta      *ResponseMeta  `json:"meta,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

// ErrorResponse represents an error in API responses
type ErrorResponse struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ResponseMeta contains metadata for API responses
type ResponseMeta struct {
	Pagination *PaginationMeta `json:"pagination,omitempty"`
	Total      int             `json:"total,omitempty"`
	Duration   string          `json:"duration,omitempty"`
}

// PaginationMeta contains pagination information
type PaginationMeta struct {
	CurrentPage int  `json:"current_page"`
	PageSize    int  `json:"page_size"`
	TotalPages  int  `json:"total_pages"`
	TotalItems  int  `json:"total_items"`
	HasNext     bool `json:"has_next"`
	HasPrev     bool `json:"has_prev"`
}

// HTTPHandler provides common HTTP handling utilities
type HTTPHandler struct {
	logger logging.Logger
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(logger logging.Logger) *HTTPHandler {
	return &HTTPHandler{
		logger: logger,
	}
}

// WriteResponse writes a standard JSON response
func (h *HTTPHandler) WriteResponse(w http.ResponseWriter, r *http.Request, statusCode int, data interface{}) {
	h.WriteResponseWithMeta(w, r, statusCode, data, nil)
}

// WriteResponseWithMeta writes a JSON response with metadata
func (h *HTTPHandler) WriteResponseWithMeta(w http.ResponseWriter, r *http.Request, statusCode int, data interface{}, meta *ResponseMeta) {
	response := StandardResponse{
		Success:   statusCode >= 200 && statusCode < 300,
		Data:      data,
		Meta:      meta,
		Timestamp: time.Now().UTC(),
	}

	h.writeJSON(w, r, statusCode, response)
}

// WriteError writes an error response
func (h *HTTPHandler) WriteError(w http.ResponseWriter, r *http.Request, err error) {
	statusCode := h.getHTTPStatusFromError(err)

	var errorResp *ErrorResponse
	if appErr, ok := err.(*errors.AppError); ok {
		errorResp = &ErrorResponse{
			Code:    string(appErr.Code),
			Message: appErr.Message,
			Details: appErr.Fields,
		}
	} else {
		errorResp = &ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: err.Error(),
		}
	}

	response := StandardResponse{
		Success:   false,
		Error:     errorResp,
		Timestamp: time.Now().UTC(),
	}

	h.writeJSON(w, r, statusCode, response)
}

// WritePaginatedResponse writes a paginated response
func (h *HTTPHandler) WritePaginatedResponse(w http.ResponseWriter, r *http.Request, data interface{}, pagination *utils.PaginationParams) {
	meta := &ResponseMeta{
		Total: pagination.Total,
		Pagination: &PaginationMeta{
			CurrentPage: pagination.Page,
			PageSize:    pagination.PageSize,
			TotalPages:  pagination.TotalPages(),
			TotalItems:  pagination.Total,
			HasNext:     pagination.HasNextPage(),
			HasPrev:     pagination.HasPrevPage(),
		},
	}

	h.WriteResponseWithMeta(w, r, http.StatusOK, data, meta)
}

// ParseQueryParams extracts common query parameters from request
func (h *HTTPHandler) ParseQueryParams(r *http.Request) (*QueryParams, error) {
	params := &QueryParams{}
	query := r.URL.Query()

	// Pagination
	if pageStr := query.Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			params.Page = page
		}
	}
	if params.Page == 0 {
		params.Page = 1
	}

	if pageSizeStr := query.Get("page_size"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 {
			params.PageSize = pageSize
		}
	}
	if params.PageSize == 0 {
		params.PageSize = 20
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}

	// Sorting
	if sort := query.Get("sort"); sort != "" {
		params.Sort = sort
	}

	if order := query.Get("order"); order != "" {
		order = strings.ToLower(order)
		if order == "asc" || order == "desc" {
			params.Order = order
		}
	}
	if params.Order == "" {
		params.Order = "asc"
	}

	// Filtering
	params.Filters = make(map[string]string)
	for key, values := range query {
		if len(values) > 0 && !isReservedParam(key) {
			params.Filters[key] = values[0]
		}
	}

	// Search
	if search := query.Get("search"); search != "" {
		params.Search = search
	}

	return params, nil
}

// QueryParams represents parsed query parameters
type QueryParams struct {
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
	Sort     string            `json:"sort"`
	Order    string            `json:"order"`
	Search   string            `json:"search"`
	Filters  map[string]string `json:"filters"`
}

// ToPaginationParams converts to utils.PaginationParams
func (q *QueryParams) ToPaginationParams() *utils.PaginationParams {
	return utils.NewPaginationParams(q.Page, q.PageSize)
}

// ParseJSONBody parses JSON request body into target struct
func (h *HTTPHandler) ParseJSONBody(r *http.Request, target interface{}) error {
	if r.Body == nil {
		return errors.InvalidInput("Request body is required", nil, nil)
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		h.logger.Warn("Failed to parse JSON body", map[string]interface{}{
			"error": err.Error(),
			"path":  r.URL.Path,
		})
		return errors.InvalidInput("Invalid JSON format", map[string]interface{}{
			"parse_error": err.Error(),
		}, err)
	}

	return nil
}

// ExtractIDFromPath extracts ID parameter from URL path
func (h *HTTPHandler) ExtractIDFromPath(r *http.Request, paramName string) (uint, error) {
	// This would typically work with a router like Gorilla Mux or Chi
	// For now, we'll extract from URL path segments
	pathSegments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	// Look for the parameter in path
	for i, segment := range pathSegments {
		if segment == paramName && i+1 < len(pathSegments) {
			if id, err := strconv.ParseUint(pathSegments[i+1], 10, 32); err == nil {
				return uint(id), nil
			}
		}
	}

	return 0, errors.InvalidInput(fmt.Sprintf("Invalid or missing %s parameter", paramName), nil, nil)
}

// ValidateContentType validates request content type
func (h *HTTPHandler) ValidateContentType(r *http.Request, expectedType string) error {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return errors.InvalidInput("Content-Type header is required", nil, nil)
	}

	// Parse content type (ignore charset, etc.)
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = contentType[:idx]
	}
	contentType = strings.TrimSpace(contentType)

	if contentType != expectedType {
		return errors.InvalidInput("Invalid Content-Type", map[string]interface{}{
			"expected": expectedType,
			"received": contentType,
		}, nil)
	}

	return nil
}

// SetSecurityHeaders sets common security headers
func (h *HTTPHandler) SetSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Content-Security-Policy", "default-src 'self'")
}

// SetCORSHeaders sets CORS headers for API responses
func (h *HTTPHandler) SetCORSHeaders(w http.ResponseWriter, allowedOrigins []string, allowedMethods []string) {
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}
	if len(allowedMethods) == 0 {
		allowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}

	w.Header().Set("Access-Control-Allow-Origin", strings.Join(allowedOrigins, ","))
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, ","))
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
	w.Header().Set("Access-Control-Max-Age", "86400")
}

// HandlePreflight handles CORS preflight requests
func (h *HTTPHandler) HandlePreflight(w http.ResponseWriter, r *http.Request, allowedOrigins []string, allowedMethods []string) {
	h.SetCORSHeaders(w, allowedOrigins, allowedMethods)
	w.WriteHeader(http.StatusOK)
}

// writeJSON writes JSON response with proper headers
func (h *HTTPHandler) writeJSON(w http.ResponseWriter, r *http.Request, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	h.SetSecurityHeaders(w)

	// Add request ID if present
	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		w.Header().Set("X-Request-ID", requestID)
	}

	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", map[string]interface{}{
			"error": err.Error(),
			"path":  r.URL.Path,
		})
	}
}

// getHTTPStatusFromError converts application errors to HTTP status codes
func (h *HTTPHandler) getHTTPStatusFromError(err error) int {
	if appErr, ok := err.(*errors.AppError); ok {
		switch appErr.Code {
		case errors.ErrNotFound:
			return http.StatusNotFound
		case errors.ErrInvalidInput:
			return http.StatusBadRequest
		case errors.ErrUnauthorized:
			return http.StatusUnauthorized
		case errors.ErrForbidden:
			return http.StatusForbidden
		case errors.ErrConflict:
			return http.StatusConflict
		case errors.ErrTimeout:
			return http.StatusRequestTimeout
		case errors.ErrUnavailable:
			return http.StatusServiceUnavailable
		default:
			return http.StatusInternalServerError
		}
	}
	return http.StatusInternalServerError
}

// isReservedParam checks if a query parameter is reserved for pagination/sorting
func isReservedParam(param string) bool {
	reserved := []string{"page", "page_size", "sort", "order", "search"}
	for _, r := range reserved {
		if param == r {
			return true
		}
	}
	return false
}

// Helper functions for common HTTP operations

// RedirectHandler creates a simple redirect handler
func RedirectHandler(url string, permanent bool) http.HandlerFunc {
	code := http.StatusFound
	if permanent {
		code = http.StatusMovedPermanently
	}
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, url, code)
	}
}

// NotFoundHandler creates a standard 404 handler
func NotFoundHandler(logger logging.Logger) http.HandlerFunc {
	handler := NewHTTPHandler(logger)
	return func(w http.ResponseWriter, r *http.Request) {
		err := errors.NotFound("Resource not found", map[string]interface{}{
			"path":   r.URL.Path,
			"method": r.Method,
		})
		handler.WriteError(w, r, err)
	}
}

// MethodNotAllowedHandler creates a standard 405 handler
func MethodNotAllowedHandler(logger logging.Logger, allowedMethods []string) http.HandlerFunc {
	handler := NewHTTPHandler(logger)
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
		err := errors.InvalidInput("Method not allowed", map[string]interface{}{
			"method":          r.Method,
			"allowed_methods": allowedMethods,
		}, nil)
		handler.WriteError(w, r, err)
	}
}

// HealthCheckHandler creates a simple health check handler
func HealthCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}
