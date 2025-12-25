package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"recommend_engine/internal/history"
	"recommend_engine/internal/model"
	taskpkg "recommend_engine/internal/task" // 使用别名导入以避免命名冲突
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
	taskManager  *taskpkg.Manager // 使用别名
}

// NewServer 创建新的 HTTP 服务器
func NewServer(up user.Provider, engine *workflow.Engine, hs history.Store, tm *taskpkg.Manager) *Server {
	s := &Server{
		router:       gin.Default(),
		userProvider: up,
		engine:       engine,
		historyStore: hs,
		taskManager:  tm, // 使用别名
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
	// 异步任务结果查询接口
	v1.GET("/recommend/result/:task_id", s.handleGetResult)
}

// handleGetResult 处理获取异步任务结果的请求
// GET /api/v1/recommend/result/:task_id
func (s *Server) handleGetResult(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task_id parameter is required"})
		return
	}

	task, err := s.taskManager.GetTask(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// 根据任务状态返回不同的响应
	switch task.Status {
	case "completed":
		c.JSON(http.StatusOK, gin.H{
			"status": task.Status,
			"data":   task.Result,
		})
	case "failed":
		c.JSON(http.StatusOK, gin.H{
			"status": task.Status,
			"error":  task.Error,
		})
	default: // "pending" or "processing"
		c.JSON(http.StatusOK, gin.H{
			"status": task.Status,
		})
	}
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
		Favorites: req.Favorites,
	}

	// 5. 检查是同步还是异步执行
	isAsync := c.Query("async") == "true"

	if isAsync {
		// --- 异步执行路径 ---
		task := s.taskManager.NewTask()
		c.JSON(http.StatusAccepted, gin.H{"task_id": task.ID})

		go func() {
			// 更新任务状态为处理中
			s.taskManager.UpdateStatus(task.ID, taskpkg.StatusProcessing)

			// 5.1 准备 Workflow Context (后台)
			// 使用独立的后台 context，并设置一个较长的超时时间（例如5分钟）作为兜底
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			wfCtx := workflow.NewContext(ctx, requestUser.ID, requestUser)
			wfCtx.Config = map[string]interface{}{"domain": scene}

			// 6. 执行推荐 (后台)
			if err := s.engine.Run(wfCtx, scene); err != nil {
				s.taskManager.SetError(task.ID, err)
				return
			}

			candidates := wfCtx.GetCandidates()

			// 7. 异步保存历史 (后台)
			var itemNames []string
			for _, item := range candidates {
				itemNames = append(itemNames, item.Name)
			}
			if len(itemNames) > 0 {
				if err := s.historyStore.SaveHistory(u.ID, scene, itemNames); err != nil {
					// 即使历史保存失败，也应将推荐结果标记为成功
					fmt.Printf("Warning: Failed to save history for task %s: %v\n", task.ID, err)
				}
			}
			
			// 8. 将最终结果存入任务
			s.taskManager.SetResult(task.ID, gin.H{
				"scene": scene,
				"items": candidates,
			})
		}()
	} else {
		// --- 同步执行路径 (保持原有逻辑不变) ---
		// 5. 准备 Workflow Context
		ctx, cancel := context.WithTimeout(c.Request.Context(), 300*time.Second)
		defer cancel()

		wfCtx := workflow.NewContext(ctx, requestUser.ID, requestUser)
		wfCtx.Config = map[string]interface{}{"domain": scene}

		// 6. 执行推荐
		if err := s.engine.Run(wfCtx, scene); err != nil {
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
}
