package handlers

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/catalog"
)

// CatalogMultiUserHandler handles multi-user catalog operations
type CatalogMultiUserHandler struct {
	service *catalog.MultiUserCatalogService
}

// CreateCatalogMultiUserHandler creates a new multi-user catalog handler
func CreateCatalogMultiUserHandler(
	service *catalog.MultiUserCatalogService,
) *CatalogMultiUserHandler {
	return &CatalogMultiUserHandler{
		service: service,
	}
}

// GetResolvedCatalog returns the resolved catalog for the current user
// GET /api/v1/catalogs/resolved
func (h *CatalogMultiUserHandler) GetResolvedCatalog(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		RespondError(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	resolved, err := h.service.GetResolvedCatalogForUser(c.Request.Context(), userID)
	if err != nil {
		RespondError(
			c,
			http.StatusInternalServerError,
			fmt.Sprintf("failed to resolve catalog: %v", err),
		)
		return
	}

	RespondJSON(c, http.StatusOK, resolved)
}

// GetUserCustomizations returns the user's catalog customizations
// GET /api/v1/catalogs/customizations
func (h *CatalogMultiUserHandler) GetUserCustomizations(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		RespondError(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	customizations, err := h.service.GetUserCustomizations(c.Request.Context(), userID)
	if err != nil {
		RespondError(
			c,
			http.StatusInternalServerError,
			fmt.Sprintf("failed to get customizations: %v", err),
		)
		return
	}

	RespondJSON(c, http.StatusOK, customizations)
}

// UpdateUserCustomizations updates the user's catalog customizations
// PUT /api/v1/catalogs/customizations
func (h *CatalogMultiUserHandler) UpdateUserCustomizations(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		RespondError(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req catalog.UserCatalogCustomizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	userCatalog, err := h.service.UpdateUserCatalogCustomization(c.Request.Context(), userID, &req)
	if err != nil {
		RespondError(
			c,
			http.StatusInternalServerError,
			fmt.Sprintf("failed to update customizations: %v", err),
		)
		return
	}

	RespondJSON(c, http.StatusOK, userCatalog)
}

// ExportUserCatalog exports the user's resolved catalog
// GET /api/v1/catalogs/export?format=json|yaml
func (h *CatalogMultiUserHandler) ExportUserCatalog(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		RespondError(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	format := c.DefaultQuery("format", "json")
	if format != "json" && format != "yaml" {
		RespondError(c, http.StatusBadRequest, "invalid format, must be json or yaml")
		return
	}

	data, err := h.service.ExportUserCatalog(c.Request.Context(), userID, format)
	if err != nil {
		RespondError(
			c,
			http.StatusInternalServerError,
			fmt.Sprintf("failed to export catalog: %v", err),
		)
		return
	}

	// Set appropriate content type
	contentType := "application/json"
	if format == "yaml" {
		contentType = "application/x-yaml"
	}

	c.Data(http.StatusOK, contentType, data)
}

// Admin endpoints - require admin role

// CreateAdminBaseCatalog creates an admin-controlled base catalog
// POST /api/v1/admin/catalogs/base
func (h *CatalogMultiUserHandler) CreateAdminBaseCatalog(c *gin.Context) {
	userID := c.GetString("user_id")
	userRole := c.GetString("user_role")

	if userID == "" || userRole != "admin" {
		RespondError(c, http.StatusForbidden, "admin access required")
		return
	}

	var req catalog.CreateCatalogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	catalog, err := h.service.CreateAdminBaseCatalog(c.Request.Context(), userID, &req)
	if err != nil {
		RespondError(
			c,
			http.StatusInternalServerError,
			fmt.Sprintf("failed to create base catalog: %v", err),
		)
		return
	}

	RespondJSON(c, http.StatusCreated, catalog)
}

// GetAdminBaseCatalogs returns all admin-controlled base catalogs
// GET /api/v1/admin/catalogs/base
func (h *CatalogMultiUserHandler) GetAdminBaseCatalogs(c *gin.Context) {
	userID := c.GetString("user_id")
	userRole := c.GetString("user_role")

	if userID == "" || userRole != "admin" {
		RespondError(c, http.StatusForbidden, "admin access required")
		return
	}

	catalogs, err := h.service.GetAdminBaseCatalogs(c.Request.Context(), userID)
	if err != nil {
		RespondError(
			c,
			http.StatusInternalServerError,
			fmt.Sprintf("failed to get base catalogs: %v", err),
		)
		return
	}

	RespondJSON(c, http.StatusOK, catalogs)
}

// UpdateAdminBaseCatalog updates an admin base catalog
// PUT /api/v1/admin/catalogs/base/:id
func (h *CatalogMultiUserHandler) UpdateAdminBaseCatalog(c *gin.Context) {
	userID := c.GetString("user_id")
	userRole := c.GetString("user_role")

	if userID == "" || userRole != "admin" {
		RespondError(c, http.StatusForbidden, "admin access required")
		return
	}

	catalogID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondError(c, http.StatusBadRequest, "invalid catalog ID")
		return
	}

	var req catalog.UpdateCatalogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	// Update catalog using base service
	catalog, err := h.service.UpdateCatalog(c.Request.Context(), userID, catalogID, &req)
	if err != nil {
		RespondError(
			c,
			http.StatusInternalServerError,
			fmt.Sprintf("failed to update base catalog: %v", err),
		)
		return
	}

	// Save updated catalog to file system
	if err := h.service.FileManager.SaveBaseCatalog(c.Request.Context(), catalog); err != nil {
		RespondError(
			c,
			http.StatusInternalServerError,
			fmt.Sprintf("failed to save base catalog: %v", err),
		)
		return
	}

	// Clear all user catalog caches
	h.service.Inheritance.ClearCache()

	RespondJSON(c, http.StatusOK, catalog)
}

// DeleteAdminBaseCatalog deletes an admin base catalog
// DELETE /api/v1/admin/catalogs/base/:id
func (h *CatalogMultiUserHandler) DeleteAdminBaseCatalog(c *gin.Context) {
	userID := c.GetString("user_id")
	userRole := c.GetString("user_role")

	if userID == "" || userRole != "admin" {
		RespondError(c, http.StatusForbidden, "admin access required")
		return
	}

	catalogID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondError(c, http.StatusBadRequest, "invalid catalog ID")
		return
	}

	// Delete from repository
	if err := h.service.DeleteCatalog(c.Request.Context(), userID, catalogID); err != nil {
		RespondError(
			c,
			http.StatusInternalServerError,
			fmt.Sprintf("failed to delete catalog: %v", err),
		)
		return
	}

	// Clear all user catalog caches
	h.service.Inheritance.ClearCache()

	RespondJSON(c, http.StatusOK, gin.H{"message": "catalog deleted successfully"})
}

// ImportAdminCatalog imports a catalog as an admin base catalog
// POST /api/v1/admin/catalogs/import
func (h *CatalogMultiUserHandler) ImportAdminCatalog(c *gin.Context) {
	userID := c.GetString("user_id")
	userRole := c.GetString("user_role")

	if userID == "" || userRole != "admin" {
		RespondError(c, http.StatusForbidden, "admin access required")
		return
	}

	// Get format from query parameter
	format := c.DefaultQuery("format", "json")
	if format != "json" && format != "yaml" {
		RespondError(c, http.StatusBadRequest, "invalid format, must be json or yaml")
		return
	}

	// Read request body
	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		RespondError(c, http.StatusBadRequest, fmt.Sprintf("failed to read request body: %v", err))
		return
	}

	catalog, err := h.service.ImportAdminCatalog(c.Request.Context(), userID, data, format)
	if err != nil {
		RespondError(
			c,
			http.StatusInternalServerError,
			fmt.Sprintf("failed to import catalog: %v", err),
		)
		return
	}

	RespondJSON(c, http.StatusCreated, catalog)
}

// GetUserCatalogStats returns statistics about catalog usage
// GET /api/v1/catalogs/stats
func (h *CatalogMultiUserHandler) GetUserCatalogStats(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		RespondError(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get resolved catalog for stats
	resolved, err := h.service.GetResolvedCatalogForUser(c.Request.Context(), userID)
	if err != nil {
		RespondError(
			c,
			http.StatusInternalServerError,
			fmt.Sprintf("failed to get catalog stats: %v", err),
		)
		return
	}

	stats := gin.H{
		"total_servers":    len(resolved.MergedCatalog.Registry),
		"admin_servers":    resolved.AdminServers,
		"user_overrides":   resolved.UserOverrides,
		"custom_servers":   resolved.CustomServers,
		"disabled_servers": len(resolved.MergedCatalog.DisabledServers),
		"last_updated":     resolved.Timestamp,
	}

	RespondJSON(c, http.StatusOK, stats)
}

// RegisterMultiUserRoutes registers the multi-user catalog routes
func RegisterMultiUserRoutes(router *gin.RouterGroup, handler *CatalogMultiUserHandler) {
	// User endpoints
	userGroup := router.Group("/catalogs")
	{
		userGroup.GET("/resolved", handler.GetResolvedCatalog)
		userGroup.GET("/customizations", handler.GetUserCustomizations)
		userGroup.PUT("/customizations", handler.UpdateUserCustomizations)
		userGroup.GET("/export", handler.ExportUserCatalog)
		userGroup.GET("/stats", handler.GetUserCatalogStats)
	}

	// Admin endpoints
	adminGroup := router.Group("/admin/catalogs")
	{
		adminGroup.POST("/base", handler.CreateAdminBaseCatalog)
		adminGroup.GET("/base", handler.GetAdminBaseCatalogs)
		adminGroup.PUT("/base/:id", handler.UpdateAdminBaseCatalog)
		adminGroup.DELETE("/base/:id", handler.DeleteAdminBaseCatalog)
		adminGroup.POST("/import", handler.ImportAdminCatalog)
	}
}

// Helper functions that should be in a shared package

// RespondJSON sends a JSON response
func RespondJSON(c *gin.Context, status int, data interface{}) {
	c.JSON(status, data)
}

// RespondError sends an error response
func RespondError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}
