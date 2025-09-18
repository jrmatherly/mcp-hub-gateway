// Package handlers provides HTTP request handlers for the MCP Portal API.
// It includes handlers for authentication, server management, gateway control,
// and other portal functionality with proper error handling and response formatting.
package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/auth"
)

// APIResponse represents a standardized API response structure
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	Meta      *Meta       `json:"meta,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// APIError represents an API error response
type APIError struct {
	Code       string            `json:"code"`
	Message    string            `json:"message"`
	Details    map[string]string `json:"details,omitempty"`
	StatusCode int               `json:"-"`
	RequestID  string            `json:"request_id,omitempty"`
}

// Meta represents response metadata
type Meta struct {
	Total      int64  `json:"total,omitempty"`
	Page       int    `json:"page,omitempty"`
	PerPage    int    `json:"per_page,omitempty"`
	TotalPages int    `json:"total_pages,omitempty"`
	Version    string `json:"version,omitempty"`
}

// PaginationParams represents pagination parameters
type PaginationParams struct {
	Page    int `form:"page"     binding:"min=1"`
	PerPage int `form:"per_page" binding:"min=1,max=100"`
}

// DefaultPaginationParams returns default pagination parameters
func DefaultPaginationParams() PaginationParams {
	return PaginationParams{
		Page:    1,
		PerPage: 20,
	}
}

// SuccessResponse sends a successful response
func SuccessResponse(c *gin.Context, data interface{}) {
	requestID, _ := c.Get("request_id")

	response := APIResponse{
		Success:   true,
		Data:      data,
		RequestID: getStringValue(requestID),
		Timestamp: time.Now().UTC(),
	}

	c.JSON(http.StatusOK, response)
}

// SuccessResponseWithMeta sends a successful response with metadata
func SuccessResponseWithMeta(c *gin.Context, data interface{}, meta *Meta) {
	requestID, _ := c.Get("request_id")

	response := APIResponse{
		Success:   true,
		Data:      data,
		Meta:      meta,
		RequestID: getStringValue(requestID),
		Timestamp: time.Now().UTC(),
	}

	c.JSON(http.StatusOK, response)
}

// CreatedResponse sends a 201 Created response
func CreatedResponse(c *gin.Context, data interface{}) {
	requestID, _ := c.Get("request_id")

	response := APIResponse{
		Success:   true,
		Data:      data,
		RequestID: getStringValue(requestID),
		Timestamp: time.Now().UTC(),
	}

	c.JSON(http.StatusCreated, response)
}

// NoContentResponse sends a 204 No Content response
func NoContentResponse(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// ErrorResponse sends an error response
func ErrorResponse(c *gin.Context, statusCode int, code, message string) {
	requestID, _ := c.Get("request_id")

	apiError := &APIError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		RequestID:  getStringValue(requestID),
	}

	response := APIResponse{
		Success:   false,
		Error:     apiError,
		RequestID: getStringValue(requestID),
		Timestamp: time.Now().UTC(),
	}

	c.JSON(statusCode, response)
}

// ErrorResponseWithDetails sends an error response with additional details
func ErrorResponseWithDetails(
	c *gin.Context,
	statusCode int,
	code, message string,
	details map[string]string,
) {
	requestID, _ := c.Get("request_id")

	apiError := &APIError{
		Code:       code,
		Message:    message,
		Details:    details,
		StatusCode: statusCode,
		RequestID:  getStringValue(requestID),
	}

	response := APIResponse{
		Success:   false,
		Error:     apiError,
		RequestID: getStringValue(requestID),
		Timestamp: time.Now().UTC(),
	}

	c.JSON(statusCode, response)
}

// ValidationErrorResponse sends a validation error response
func ValidationErrorResponse(c *gin.Context, errors map[string]string) {
	ErrorResponseWithDetails(
		c,
		http.StatusBadRequest,
		"validation_error",
		"Invalid request data",
		errors,
	)
}

// AuthErrorResponse sends an authentication error response
func AuthErrorResponse(c *gin.Context, authErr *auth.AuthError) {
	requestID, _ := c.Get("request_id")

	apiError := &APIError{
		Code:       authErr.Code,
		Message:    authErr.Message,
		StatusCode: authErr.HTTPStatus,
		RequestID:  getStringValue(requestID),
	}

	response := APIResponse{
		Success:   false,
		Error:     apiError,
		RequestID: getStringValue(requestID),
		Timestamp: time.Now().UTC(),
	}

	c.JSON(authErr.HTTPStatus, response)
}

// NotFoundResponse sends a 404 Not Found response
func NotFoundResponse(c *gin.Context, resource string) {
	ErrorResponse(c, http.StatusNotFound, "not_found", "The requested "+resource+" was not found")
}

// ForbiddenResponse sends a 403 Forbidden response
func ForbiddenResponse(c *gin.Context, message string) {
	if message == "" {
		message = "Access denied"
	}
	ErrorResponse(c, http.StatusForbidden, "forbidden", message)
}

// UnauthorizedResponse sends a 401 Unauthorized response
func UnauthorizedResponse(c *gin.Context, message string) {
	if message == "" {
		message = "Authentication required"
	}
	ErrorResponse(c, http.StatusUnauthorized, "unauthorized", message)
}

// ConflictResponse sends a 409 Conflict response
func ConflictResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusConflict, "conflict", message)
}

// InternalErrorResponse sends a 500 Internal Server Error response
func InternalErrorResponse(c *gin.Context, message string) {
	if message == "" {
		message = "An internal error occurred"
	}
	ErrorResponse(c, http.StatusInternalServerError, "internal_error", message)
}

// RateLimitResponse sends a 429 Too Many Requests response
func RateLimitResponse(c *gin.Context, retryAfter int) {
	requestID, _ := c.Get("request_id")

	c.Header("Retry-After", string(rune(retryAfter)))

	apiError := &APIError{
		Code:       "rate_limit_exceeded",
		Message:    "Too many requests. Please try again later.",
		StatusCode: http.StatusTooManyRequests,
		RequestID:  getStringValue(requestID),
		Details: map[string]string{
			"retry_after": string(rune(retryAfter)),
		},
	}

	response := APIResponse{
		Success:   false,
		Error:     apiError,
		RequestID: getStringValue(requestID),
		Timestamp: time.Now().UTC(),
	}

	c.JSON(http.StatusTooManyRequests, response)
}

// GetUserFromContext extracts the authenticated user from the request context
func GetUserFromContext(c *gin.Context) (*auth.User, bool) {
	return auth.GetUserFromContext(c.Request.Context())
}

// GetRequestID extracts the request ID from the context
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get("request_id"); exists {
		return getStringValue(requestID)
	}
	return ""
}

// ValidateUUID validates a UUID parameter
func ValidateUUID(param string) (uuid.UUID, error) {
	return uuid.Parse(param)
}

// getStringValue safely converts an interface{} to string
func getStringValue(value interface{}) string {
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}

// BindPagination binds and validates pagination parameters
func BindPagination(c *gin.Context) PaginationParams {
	var params PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		// Return defaults if binding fails
		return DefaultPaginationParams()
	}

	// Set defaults if not provided
	if params.Page == 0 {
		params.Page = 1
	}
	if params.PerPage == 0 {
		params.PerPage = 20
	}

	// Enforce limits
	if params.PerPage > 100 {
		params.PerPage = 100
	}

	return params
}

// CalculateOffset calculates the offset for pagination
func CalculateOffset(page, perPage int) int {
	return (page - 1) * perPage
}

// CalculateTotalPages calculates the total number of pages
func CalculateTotalPages(total int64, perPage int) int {
	if perPage == 0 {
		return 0
	}
	return int((total + int64(perPage) - 1) / int64(perPage))
}

// CreateMeta creates a Meta object for paginated responses
func CreateMeta(total int64, page, perPage int) *Meta {
	return &Meta{
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: CalculateTotalPages(total, perPage),
	}
}
