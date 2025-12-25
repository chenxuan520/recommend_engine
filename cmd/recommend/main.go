package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"recommend_engine/internal/history"
	"recommend_engine/internal/nodes"
	"recommend_engine/internal/server"
	"recommend_engine/internal/user"
	"recommend_engine/internal/workflow"
	"recommend_engine/pkg/llm"

	"gopkg.in/yaml.v3"
)

// LLMGlobalConfig 对应 configs/llm.yaml
type LLMGlobalConfig struct {
	LLMs map[string]struct {
		ChatEndpoint string `yaml:"chat_endpoint"` // 完整的 API 地址
		APIKey       string `yaml:"api_key"`
		Model        string `yaml:"model"`
	} `yaml:"llms"`
}

func loadLLMConfig(path string) (*LLMGlobalConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg LLMGlobalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func main() {
	// 命令行参数
	port := flag.String("port", "8080", "Server port")
	userConfigPath := flag.String("users", "configs/users.yaml", "Path to users.yaml")
	pipelineConfigPath := flag.String("pipelines", "configs/pipelines.json", "Path to pipelines.json")
	llmConfigPath := flag.String("llm", "configs/llm.yaml", "Path to llm.yaml")
	historyPath := flag.String("history", "history.jsonl", "Path to history.jsonl")
	flag.Parse()

	// 1. 初始化 User Provider
	userProvider, err := user.NewStaticProvider(*userConfigPath)
	if err != nil {
		log.Fatalf("Failed to init user provider: %v", err)
	}

	// 2. 初始化 History Store
	historyStore, err := history.NewFileStore(*historyPath)
	if err != nil {
		log.Fatalf("Failed to init history store: %v", err)
	}

	// 3. 加载 LLM 配置
	llmCfg, err := loadLLMConfig(*llmConfigPath)
	if err != nil {
		log.Fatalf("Failed to load llm config: %v", err)
	}

	// 4. 初始化 Node Registry 并注册节点
	registry := workflow.NewRegistry()

	// 注册 LLM Recall
	registry.Register("recall", func(cfg workflow.NodeConfig) (workflow.Node, error) {
		// 从 Pipeline Config 获取 key
		key, ok := cfg.Config["llm_config_key"].(string)
		if !ok {
			return nil, fmt.Errorf("llm_recall_node '%s' missing 'llm_config_key'", cfg.Name)
		}

		// 从 Global Config 获取凭证
		cred, ok := llmCfg.LLMs[key]
		if !ok {
			return nil, fmt.Errorf("llm config key '%s' not found in %s", key, *llmConfigPath)
		}

		// 构造 Client
		client := llm.NewOpenAIClient(cred.ChatEndpoint, cred.APIKey, cred.Model)

		// 获取其他参数
		count, _ := cfg.Config["count"].(float64)

		return nodes.NewLLMRecallNode(cfg.Name, client, int(count)), nil
	})

	// 注册 History Filter (使用闭包注入 historyStore)
	registry.Register("filter", func(cfg workflow.NodeConfig) (workflow.Node, error) {
		return nodes.NewHistoryFilterNode(cfg, historyStore)
	})
	
	// 注册 Favorites Filter (新)
	registry.Register("filter_favorites", nodes.NewFavoritesFilterNode)

	// 注册 Rank
	registry.Register("rank", nodes.NewSimpleRankNode)
	
	// 注册 Mix Favorites Rank (新)
	registry.Register("rank_mix_favorites", nodes.NewMixFavoritesRankNode)

	// 5. 初始化 Pipeline Engine
	engine, err := workflow.NewEngine(*pipelineConfigPath, registry)
	if err != nil {
		log.Fatalf("Failed to init engine: %v", err)
	}

	// 6. 启动 HTTP Server
	srv := server.NewServer(userProvider, engine, historyStore)
	log.Printf("Starting HTTP server on port %s...", *port)
	if err := srv.Run(":" + *port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
