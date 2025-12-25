package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"recommend_engine/internal/history"
	"recommend_engine/internal/model"
	"recommend_engine/internal/user"
	"recommend_engine/internal/workflow"

	"github.com/gin-gonic/gin"
)

// Server 代表 HTTP API 服务器
type Server struct {
	router       *gin.Engine
	userProvider user.Provider
	engine       *workflow.Engine
	historyStore history.Store
}

// NewServer 创建新的 HTTP 服务器
func NewServer(up user.Provider, engine *workflow.Engine, hs history.Store) *Server {
	s := &Server{
		router:       gin.Default(),
		userProvider: up,
		engine:       engine,
		historyStore: hs,
	}
	s.router.Use(s.corsMiddleware())
	s.setupRoutes()
	return s
}

func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// Run 启动服务器
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

func (s *Server) setupRoutes() {
	v1 := s.router.Group("/api/v1")

	// 中间件：Token 鉴权
	v1.Use(s.authMiddleware())

	// 推荐接口 - 使用路径参数传递 scene
	v1.POST("/recommend/:scene", s.handleRecommend)
}

// authMiddleware 鉴权中间件
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}

		token := parts[1]
		u, err := s.userProvider.GetUserByToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		// 将用户信息存入 Context
		c.Set("user", u)
		c.Next()
	}
}

type RecommendRequest struct {
	// Scene     string   `json:"scene"` // 移除 Scene 字段，改用 URL Path 参数
	Favorites []string `json:"favorites" binding:"required"`
}

// handleRecommend 处理推荐请求
// POST /api/v1/recommend/:scene
func (s *Server) handleRecommend(c *gin.Context) {
	// 1. 获取路径参数
	scene := c.Param("scene")
	if scene == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "scene parameter is required"})
		return
	}

	// 2. 解析请求参数 (favorites)
	var req RecommendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	// 3. 从 Context 获取鉴权用户
	uVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	u := uVal.(*model.User)

	// 4. 构建请求级用户对象 (使用请求中的 favorites)
	requestUser := &model.User{
		ID:        u.ID,
		Name:      u.Name,
		Token:     u.Token,
		Favorites: req.Favorites, // 使用请求中的动态收藏列表
	}

	// 5. 准备 Workflow Context
	// 设置 300s 超时
	ctx, cancel := context.WithTimeout(c.Request.Context(), 300*time.Second)
	defer cancel()

	wfCtx := workflow.NewContext(ctx, requestUser.ID, requestUser)
	wfCtx.Config = map[string]interface{}{
		"domain": scene,
	}

	// 6. 执行推荐
	if err := s.engine.Run(wfCtx, scene); err != nil {
		// 区分 pipeline not found 错误和执行错误
		if strings.Contains(err.Error(), "pipeline not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("scene '%s' not supported", scene)})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("recommendation failed: %v", err)})
		return
	}

	candidates := wfCtx.GetCandidates()

	// 7. 异步保存历史
	go func() {
		var itemNames []string
		for _, item := range candidates {
			itemNames = append(itemNames, item.Name)
		}
		if len(itemNames) > 0 {
			if err := s.historyStore.SaveHistory(u.ID, scene, itemNames); err != nil {
				fmt.Printf("Failed to save history async: %v\n", err)
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"scene": scene,
		"items": candidates,
	})
}
