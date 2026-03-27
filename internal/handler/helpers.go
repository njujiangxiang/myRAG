package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Helper functions for common handler patterns

// GetTenantID extracts and validates tenant ID from gin context
func GetTenantID(c *gin.Context) (uuid.UUID, bool) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return uuid.Nil, false
	}
	u, ok := tenantID.(uuid.UUID)
	if !ok {
		return uuid.Nil, false
	}
	return u, true
}

// GetUserID extracts and validates user ID from gin context
func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, false
	}
	u, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil, false
	}
	return u, true
}

// ParseUUIDParam parses a UUID from a URL parameter
func ParseUUIDParam(c *gin.Context, paramName string) (uuid.UUID, error) {
	idStr := c.Param(paramName)
	return uuid.Parse(idStr)
}

// VerifyTenantOwnership checks if the resource belongs to the tenant
func VerifyTenantOwnership(resourceTenantID, requestTenantID uuid.UUID) bool {
	return resourceTenantID == requestTenantID
}
