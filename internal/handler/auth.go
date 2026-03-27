package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"myrag/internal/jwt"
	"myrag/internal/models"
)

// AuthHandler 处理认证请求
type AuthHandler struct {
	userRepo *models.UserRepository
	jwt      *jwt.JWT
	secret   string
}

// NewAuthHandler 创建一个新的认证处理器
func NewAuthHandler(userRepo *models.UserRepository, secret string) *AuthHandler {
	return &AuthHandler{
		userRepo: userRepo,
		jwt:      jwt.New(secret),
		secret:   secret,
	}
}

// RegisterRequest 表示注册请求
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginRequest 表示登录请求
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse 表示认证响应
type AuthResponse struct {
	Token string     `json:"token"`
	User  UserResult `json:"user"`
}

// UserResult 表示响应中的用户数据
type UserResult struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// Register 处理用户注册
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查用户是否已存在
	existing, err := h.userRepo.GetByEmail(c.Request.Context(), req.Email)
	if err == nil && existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
		return
	}

	// 哈希密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	// 获取或创建默认租户
	tenantID := uuid.Nil // Will be set by middleware or use default
	if tenantID == uuid.Nil {
		// 对于单用户模式，使用预定义的租户
		tenantID, _ = uuid.Parse("00000000-0000-0000-0000-000000000001")
	}

	// 创建用户
	user := &models.User{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Role:         "user",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := h.userRepo.Create(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	// 生成 JWT token
	token, err := h.jwt.GenerateToken(user.ID, user.TenantID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, AuthResponse{
		Token: token,
		User: UserResult{
			ID:        user.ID,
			Email:     user.Email,
			Role:      user.Role,
			CreatedAt: user.CreatedAt,
		},
	})
}

// Login 处理用户登录
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 通过邮箱获取用户
	user, err := h.userRepo.GetByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// 检查密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// 生成 JWT token
	token, err := h.jwt.GenerateToken(user.ID, user.TenantID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Token: token,
		User: UserResult{
			ID:        user.ID,
			Email:     user.Email,
			Role:      user.Role,
			CreatedAt: user.CreatedAt,
		},
	})
}

// Logout 处理用户登出
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// 对于无状态 JWT，登出是客户端行为（只需丢弃 token）
	// 可选：我们可以维护一个被撤销的 token 黑名单
	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

// Me 返回当前用户信息
// GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, UserResult{
		ID:        user.ID,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	})
}

// generateJWT 创建 JWT token
// Deprecated: use h.jwt.GenerateToken instead
func generateJWT(userID, tenantID uuid.UUID, email string, secret string) (string, error) {
	j := jwt.New(secret)
	return j.GenerateToken(userID, tenantID, email)
}

// AuthMiddleware 从请求中提取并验证 JWT
func AuthMiddleware(secret string) gin.HandlerFunc {
	j := jwt.New(secret)

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		// 从 "Bearer <token>" 格式中提取 token
		tokenString := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		}

		// 验证 JWT token 并提取声明
		claims, err := j.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// 在上下文中设置 user_id 和 tenant_id
		c.Set("user_id", claims.UserID)
		c.Set("tenant_id", claims.TenantID)
		c.Next()
	}
}

// TenantMiddleware 从用户声明中设置租户上下文
func TenantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, exists := c.Get("tenant_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context missing"})
			c.Abort()
			return
		}

		// 在数据库连接中设置租户上下文
		ctx := context.WithValue(c.Request.Context(), "tenant_id", tenantID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
