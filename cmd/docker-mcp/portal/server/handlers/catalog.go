package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/catalog"
)

// CatalogHandler handles catalog-related HTTP requests
type CatalogHandler struct {
	catalogService catalog.CatalogService
}

// CreateCatalogHandler creates a new catalog handler instance
func CreateCatalogHandler(catalogService catalog.CatalogService) *CatalogHandler {
	return &CatalogHandler{
		catalogService: catalogService,
	}
}

// CreateCatalog handles POST /api/v1/catalogs
func (h *CatalogHandler) CreateCatalog(c *gin.Context) {
	user, exists := GetUserFromContext(c)
	if !exists {
		UnauthorizedResponse(c, "Authentication required")
		return
	}

	var req catalog.CreateCatalogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationErrorResponse(c, map[string]string{
			"request": "Invalid JSON format or missing required fields",
		})
		return
	}

	catalogEntity, err := h.catalogService.CreateCatalog(
		c.Request.Context(),
		user.ID.String(),
		&req,
	)
	if err != nil {
		if validationErrs, ok := err.(*catalog.ValidationErrors); ok {
			details := make(map[string]string)
			for _, ve := range validationErrs.Errors {
				details[ve.Field] = ve.Message
			}
			ValidationErrorResponse(c, details)
			return
		}
		InternalErrorResponse(c, "Failed to create catalog")
		return
	}

	CreatedResponse(c, catalogEntity)
}

// GetCatalog handles GET /api/v1/catalogs/:id
func (h *CatalogHandler) GetCatalog(c *gin.Context) {
	user, exists := GetUserFromContext(c)
	if !exists {
		UnauthorizedResponse(c, "Authentication required")
		return
	}

	id, err := ValidateUUID(c.Param("id"))
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "invalid_id", "Invalid catalog ID format")
		return
	}

	catalogEntity, err := h.catalogService.GetCatalog(c.Request.Context(), user.ID.String(), id)
	if err != nil {
		if err == catalog.ErrCatalogNotFound {
			NotFoundResponse(c, "catalog")
			return
		}
		InternalErrorResponse(c, "Failed to retrieve catalog")
		return
	}

	SuccessResponse(c, catalogEntity)
}

// UpdateCatalog handles PUT /api/v1/catalogs/:id
func (h *CatalogHandler) UpdateCatalog(c *gin.Context) {
	user, exists := GetUserFromContext(c)
	if !exists {
		UnauthorizedResponse(c, "Authentication required")
		return
	}

	id, err := ValidateUUID(c.Param("id"))
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "invalid_id", "Invalid catalog ID format")
		return
	}

	var req catalog.UpdateCatalogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationErrorResponse(c, map[string]string{
			"request": "Invalid JSON format",
		})
		return
	}

	catalogEntity, err := h.catalogService.UpdateCatalog(
		c.Request.Context(),
		user.ID.String(),
		id,
		&req,
	)
	if err != nil {
		if err == catalog.ErrCatalogNotFound {
			NotFoundResponse(c, "catalog")
			return
		}
		if validationErrs, ok := err.(*catalog.ValidationErrors); ok {
			details := make(map[string]string)
			for _, ve := range validationErrs.Errors {
				details[ve.Field] = ve.Message
			}
			ValidationErrorResponse(c, details)
			return
		}
		InternalErrorResponse(c, "Failed to update catalog")
		return
	}

	SuccessResponse(c, catalogEntity)
}

// DeleteCatalog handles DELETE /api/v1/catalogs/:id
func (h *CatalogHandler) DeleteCatalog(c *gin.Context) {
	user, exists := GetUserFromContext(c)
	if !exists {
		UnauthorizedResponse(c, "Authentication required")
		return
	}

	id, err := ValidateUUID(c.Param("id"))
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "invalid_id", "Invalid catalog ID format")
		return
	}

	err = h.catalogService.DeleteCatalog(c.Request.Context(), user.ID.String(), id)
	if err != nil {
		if err == catalog.ErrCatalogNotFound {
			NotFoundResponse(c, "catalog")
			return
		}
		InternalErrorResponse(c, "Failed to delete catalog")
		return
	}

	NoContentResponse(c)
}

// ListCatalogs handles GET /api/v1/catalogs
func (h *CatalogHandler) ListCatalogs(c *gin.Context) {
	user, exists := GetUserFromContext(c)
	if !exists {
		UnauthorizedResponse(c, "Authentication required")
		return
	}

	// Bind pagination parameters
	pagination := BindPagination(c)

	// Bind filter parameters
	var filter catalog.CatalogFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		ValidationErrorResponse(c, map[string]string{
			"filter": "Invalid filter parameters",
		})
		return
	}

	// Set pagination in filter
	filter.Limit = pagination.PerPage
	filter.Offset = CalculateOffset(pagination.Page, pagination.PerPage)

	catalogs, total, err := h.catalogService.ListCatalogs(
		c.Request.Context(),
		user.ID.String(),
		filter,
	)
	if err != nil {
		InternalErrorResponse(c, "Failed to list catalogs")
		return
	}

	meta := CreateMeta(total, pagination.Page, pagination.PerPage)
	SuccessResponseWithMeta(c, catalogs, meta)
}

// SyncCatalog handles POST /api/v1/catalogs/:id/sync
func (h *CatalogHandler) SyncCatalog(c *gin.Context) {
	user, exists := GetUserFromContext(c)
	if !exists {
		UnauthorizedResponse(c, "Authentication required")
		return
	}

	id, err := ValidateUUID(c.Param("id"))
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "invalid_id", "Invalid catalog ID format")
		return
	}

	var req catalog.CatalogSyncRequest
	req.CatalogID = id

	// Parse optional sync parameters
	if c.Query("force") == "true" {
		req.Force = true
	}

	result, err := h.catalogService.SyncCatalog(c.Request.Context(), user.ID.String(), &req)
	if err != nil {
		if err == catalog.ErrCatalogNotFound {
			NotFoundResponse(c, "catalog")
			return
		}
		InternalErrorResponse(c, "Failed to start catalog sync")
		return
	}

	SuccessResponse(c, result)
}

// ImportCatalog handles POST /api/v1/catalogs/import
func (h *CatalogHandler) ImportCatalog(c *gin.Context) {
	user, exists := GetUserFromContext(c)
	if !exists {
		UnauthorizedResponse(c, "Authentication required")
		return
	}

	var req catalog.CatalogImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationErrorResponse(c, map[string]string{
			"request": "Invalid JSON format or missing required fields",
		})
		return
	}

	catalogEntity, err := h.catalogService.ImportCatalog(
		c.Request.Context(),
		user.ID.String(),
		&req,
	)
	if err != nil {
		if validationErrs, ok := err.(*catalog.ValidationErrors); ok {
			details := make(map[string]string)
			for _, ve := range validationErrs.Errors {
				details[ve.Field] = ve.Message
			}
			ValidationErrorResponse(c, details)
			return
		}
		InternalErrorResponse(c, "Failed to import catalog")
		return
	}

	CreatedResponse(c, catalogEntity)
}

// ExportCatalog handles GET /api/v1/catalogs/:id/export
func (h *CatalogHandler) ExportCatalog(c *gin.Context) {
	user, exists := GetUserFromContext(c)
	if !exists {
		UnauthorizedResponse(c, "Authentication required")
		return
	}

	id, err := ValidateUUID(c.Param("id"))
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "invalid_id", "Invalid catalog ID format")
		return
	}

	req := catalog.CatalogExportRequest{
		CatalogID: id,
		Format:    c.DefaultQuery("format", "json"),
		Minify:    c.Query("minify") == "true",
	}

	if c.Query("include_stats") == "true" {
		req.IncludeStats = true
	}

	data, err := h.catalogService.ExportCatalog(c.Request.Context(), user.ID.String(), &req)
	if err != nil {
		if err == catalog.ErrCatalogNotFound {
			NotFoundResponse(c, "catalog")
			return
		}
		InternalErrorResponse(c, "Failed to export catalog")
		return
	}

	// Set appropriate content type based on format
	switch req.Format {
	case "yaml":
		c.Data(http.StatusOK, "application/x-yaml", data)
	default:
		c.Data(http.StatusOK, "application/json", data)
	}
}

// ForkCatalog handles POST /api/v1/catalogs/:id/fork
func (h *CatalogHandler) ForkCatalog(c *gin.Context) {
	user, exists := GetUserFromContext(c)
	if !exists {
		UnauthorizedResponse(c, "Authentication required")
		return
	}

	sourceID, err := ValidateUUID(c.Param("id"))
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "invalid_id", "Invalid catalog ID format")
		return
	}

	var req struct {
		Name string `json:"name" binding:"required,min=1,max=100"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationErrorResponse(c, map[string]string{
			"name": "Fork name is required",
		})
		return
	}

	fork, err := h.catalogService.ForkCatalog(
		c.Request.Context(),
		user.ID.String(),
		sourceID,
		req.Name,
	)
	if err != nil {
		if err == catalog.ErrCatalogNotFound {
			NotFoundResponse(c, "catalog")
			return
		}
		InternalErrorResponse(c, "Failed to fork catalog")
		return
	}

	CreatedResponse(c, fork)
}

// CreateServer handles POST /api/v1/catalogs/:id/servers
func (h *CatalogHandler) CreateServer(c *gin.Context) {
	user, exists := GetUserFromContext(c)
	if !exists {
		UnauthorizedResponse(c, "Authentication required")
		return
	}

	catalogID, err := ValidateUUID(c.Param("id"))
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "invalid_id", "Invalid catalog ID format")
		return
	}

	var req catalog.CreateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationErrorResponse(c, map[string]string{
			"request": "Invalid JSON format or missing required fields",
		})
		return
	}

	req.CatalogID = catalogID

	server, err := h.catalogService.CreateServer(c.Request.Context(), user.ID.String(), &req)
	if err != nil {
		if validationErrs, ok := err.(*catalog.ValidationErrors); ok {
			details := make(map[string]string)
			for _, ve := range validationErrs.Errors {
				details[ve.Field] = ve.Message
			}
			ValidationErrorResponse(c, details)
			return
		}
		InternalErrorResponse(c, "Failed to create server")
		return
	}

	CreatedResponse(c, server)
}

// GetServer handles GET /api/v1/servers/:id
func (h *CatalogHandler) GetServer(c *gin.Context) {
	user, exists := GetUserFromContext(c)
	if !exists {
		UnauthorizedResponse(c, "Authentication required")
		return
	}

	id, err := ValidateUUID(c.Param("id"))
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "invalid_id", "Invalid server ID format")
		return
	}

	server, err := h.catalogService.GetServer(c.Request.Context(), user.ID.String(), id)
	if err != nil {
		if err == catalog.ErrServerNotFound {
			NotFoundResponse(c, "server")
			return
		}
		InternalErrorResponse(c, "Failed to retrieve server")
		return
	}

	SuccessResponse(c, server)
}

// UpdateServer handles PUT /api/v1/servers/:id
func (h *CatalogHandler) UpdateServer(c *gin.Context) {
	user, exists := GetUserFromContext(c)
	if !exists {
		UnauthorizedResponse(c, "Authentication required")
		return
	}

	id, err := ValidateUUID(c.Param("id"))
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "invalid_id", "Invalid server ID format")
		return
	}

	var req catalog.UpdateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationErrorResponse(c, map[string]string{
			"request": "Invalid JSON format",
		})
		return
	}

	server, err := h.catalogService.UpdateServer(c.Request.Context(), user.ID.String(), id, &req)
	if err != nil {
		if err == catalog.ErrServerNotFound {
			NotFoundResponse(c, "server")
			return
		}
		if validationErrs, ok := err.(*catalog.ValidationErrors); ok {
			details := make(map[string]string)
			for _, ve := range validationErrs.Errors {
				details[ve.Field] = ve.Message
			}
			ValidationErrorResponse(c, details)
			return
		}
		InternalErrorResponse(c, "Failed to update server")
		return
	}

	SuccessResponse(c, server)
}

// DeleteServer handles DELETE /api/v1/servers/:id
func (h *CatalogHandler) DeleteServer(c *gin.Context) {
	user, exists := GetUserFromContext(c)
	if !exists {
		UnauthorizedResponse(c, "Authentication required")
		return
	}

	id, err := ValidateUUID(c.Param("id"))
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "invalid_id", "Invalid server ID format")
		return
	}

	err = h.catalogService.DeleteServer(c.Request.Context(), user.ID.String(), id)
	if err != nil {
		if err == catalog.ErrServerNotFound {
			NotFoundResponse(c, "server")
			return
		}
		InternalErrorResponse(c, "Failed to delete server")
		return
	}

	NoContentResponse(c)
}

// ListServers handles GET /api/v1/servers
func (h *CatalogHandler) ListServers(c *gin.Context) {
	user, exists := GetUserFromContext(c)
	if !exists {
		UnauthorizedResponse(c, "Authentication required")
		return
	}

	// Bind pagination parameters
	pagination := BindPagination(c)

	// Bind filter parameters
	var filter catalog.ServerFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		ValidationErrorResponse(c, map[string]string{
			"filter": "Invalid filter parameters",
		})
		return
	}

	// Handle catalog_id filter from query param or path
	if catalogIDStr := c.Query("catalog_id"); catalogIDStr != "" {
		if catalogID, err := uuid.Parse(catalogIDStr); err == nil {
			filter.CatalogID = &catalogID
		}
	}

	// Set pagination in filter
	filter.Limit = pagination.PerPage
	filter.Offset = CalculateOffset(pagination.Page, pagination.PerPage)

	servers, total, err := h.catalogService.ListServers(
		c.Request.Context(),
		user.ID.String(),
		filter,
	)
	if err != nil {
		InternalErrorResponse(c, "Failed to list servers")
		return
	}

	meta := CreateMeta(total, pagination.Page, pagination.PerPage)
	SuccessResponseWithMeta(c, servers, meta)
}

// ExecuteCatalogCommand handles POST /api/v1/catalogs/command
func (h *CatalogHandler) ExecuteCatalogCommand(c *gin.Context) {
	user, exists := GetUserFromContext(c)
	if !exists {
		UnauthorizedResponse(c, "Authentication required")
		return
	}

	var req struct {
		Command string   `json:"command" binding:"required"`
		Args    []string `json:"args"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationErrorResponse(c, map[string]string{
			"command": "Command is required",
		})
		return
	}

	result, err := h.catalogService.ExecuteCatalogCommand(
		c.Request.Context(),
		user.ID.String(),
		req.Command,
		req.Args,
	)
	if err != nil {
		InternalErrorResponse(c, "Failed to execute catalog command")
		return
	}

	SuccessResponse(c, result)
}

// RegisterCatalogRoutes registers all catalog-related routes
func RegisterCatalogRoutes(r *gin.RouterGroup, handler *CatalogHandler) {
	// Catalog routes
	catalogs := r.Group("/catalogs")
	{
		catalogs.POST("", handler.CreateCatalog)
		catalogs.GET("", handler.ListCatalogs)
		catalogs.POST("/import", handler.ImportCatalog)
		catalogs.POST("/command", handler.ExecuteCatalogCommand)

		catalogs.GET("/:id", handler.GetCatalog)
		catalogs.PUT("/:id", handler.UpdateCatalog)
		catalogs.DELETE("/:id", handler.DeleteCatalog)
		catalogs.POST("/:id/sync", handler.SyncCatalog)
		catalogs.GET("/:id/export", handler.ExportCatalog)
		catalogs.POST("/:id/fork", handler.ForkCatalog)
		catalogs.POST("/:id/servers", handler.CreateServer)
	}

	// Server routes
	servers := r.Group("/servers")
	{
		servers.GET("", handler.ListServers)
		servers.GET("/:id", handler.GetServer)
		servers.PUT("/:id", handler.UpdateServer)
		servers.DELETE("/:id", handler.DeleteServer)
	}
}
