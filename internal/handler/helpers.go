package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// 辅助函数，用于常见的 handler 模式

// GetTenantID 从 gin 上下文中提取并验证租户 ID
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

// GetUserID 从 gin 上下文中提取并验证用户 ID
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

// ParseUUIDParam 从 URL 参数解析 UUID
func ParseUUIDParam(c *gin.Context, paramName string) (uuid.UUID, error) {
	idStr := c.Param(paramName)
	return uuid.Parse(idStr)
}

// VerifyTenantOwnership 检查资源是否属于该租户
func VerifyTenantOwnership(resourceTenantID, requestTenantID uuid.UUID) bool {
	return resourceTenantID == requestTenantID
}
