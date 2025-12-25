package main

import (
	"log"

	"recommend_engine/internal/history"
	"recommend_engine/internal/logger"
	"recommend_engine/internal/server"
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

	// 7. 启动 HTTP Server
	srv := server.NewServer(userProvider, engine, historyStore)
	log.Printf("Starting HTTP server on port %s...", serverCfg.Server.Port)
	if err := srv.Run(":" + serverCfg.Server.Port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
