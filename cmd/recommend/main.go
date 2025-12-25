package main

import (
	"log"
	"time"

	"recommend_engine/internal/history"
	"recommend_engine/internal/logger"
	"recommend_engine/internal/server"
	taskpkg "recommend_engine/internal/task" // 使用别名导入
	"recommend_engine/internal/user"
	"recommend_engine/internal/workflow"
)

func main() {
	// 1. 初始化并加载配置
	serverCfg := InitServerConfig()

	// 设置日志级别
	logger.SetDebug(serverCfg.Server.Debug)
	if serverCfg.Server.Debug {
		logger.Info("Debug logging enabled")
	}

	// 2. 初始化 User Provider
	userProvider, err := user.NewStaticProvider(serverCfg.Paths.Users)
	if err != nil {
		log.Fatalf("Failed to init user provider: %v", err)
	}

	// 3. 初始化 History Store
	historyStore, err := history.NewFileStore(serverCfg.Paths.History)
	if err != nil {
		log.Fatalf("Failed to init history store: %v", err)
	}

	// 自动清理过期历史记录 (保留7天)
	// 启动时先执行一次，释放内存并瘦身文件
	if err := historyStore.Cleanup(7); err != nil {
		log.Printf("Warning: Failed to perform initial history cleanup: %v", err)
	}
	// 启动定时清理任务 (每24小时)
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if err := historyStore.Cleanup(7); err != nil {
				log.Printf("Error during scheduled history cleanup: %v", err)
			} else {
				log.Println("History cleanup completed successfully")
			}
		}
	}()

	// 4. 加载 LLM 配置
	llmCfg, err := loadLLMConfig(serverCfg.Paths.LLM)
	if err != nil {
		log.Fatalf("Failed to load llm config: %v", err)
	}

	// 5. 初始化 Node Registry 并注册节点
	registry := RegisterNodes(llmCfg, serverCfg.Paths.LLM, historyStore)

	// 6. 初始化 Pipeline Engine
	engine, err := workflow.NewEngine(serverCfg.Paths.Pipelines, registry)
	if err != nil {
		log.Fatalf("Failed to init engine: %v", err)
	}

	// 7. 初始化 Task Manager
	taskManager := taskpkg.NewManager()

	// 8. 启动 HTTP Server
	srv := server.NewServer(userProvider, engine, historyStore, taskManager)
	log.Printf("Starting HTTP server on port %s...", serverCfg.Server.Port)
	if err := srv.Run(":" + serverCfg.Server.Port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
