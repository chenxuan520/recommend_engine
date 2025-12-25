package main

import (
	"fmt"
	"recommend_engine/internal/history"
	"recommend_engine/internal/nodes"
	"recommend_engine/internal/workflow"
	"recommend_engine/pkg/llm"
)

// RegisterNodes 注册所有可用的 Workflow 节点
func RegisterNodes(llmCfg *LLMGlobalConfig, llmConfigPath string, historyStore history.Store) *workflow.Registry {
	registry := workflow.NewRegistry()

	// 注册 LLM Recall
	registry.Register("recall_llm", func(cfg workflow.NodeConfig) (workflow.Node, error) {
		// 从 Pipeline Config 获取 key
		key, ok := cfg.Config["llm_config_key"].(string)
		if !ok {
			return nil, fmt.Errorf("llm_recall_node '%s' missing 'llm_config_key'", cfg.Name)
		}

		// 从 Global Config 获取凭证
		cred, ok := llmCfg.LLMs[key]
		if !ok {
			return nil, fmt.Errorf("llm config key '%s' not found in %s", key, llmConfigPath)
		}

		// 构造 Client
		client := llm.NewOpenAIClient(cred.ChatEndpoint, cred.APIKey, cred.Model)

		// 获取其他参数
		count, _ := cfg.Config["count"].(float64)

		return nodes.NewLLMRecallNode(cfg.Name, client, int(count)), nil
	})

	// 注册 History Filter (使用闭包注入 historyStore)
	registry.Register("filter_history", func(cfg workflow.NodeConfig) (workflow.Node, error) {
		return nodes.NewHistoryFilterNode(cfg, historyStore)
	})

	// 注册 Favorites Filter (新)
	registry.Register("filter_favorites", nodes.NewFavoritesFilterNode)

	// 注册 Rank
	registry.Register("rank_simple", nodes.NewSimpleRankNode)

	// 注册 Mix Favorites Rank (新)
	registry.Register("rank_mix_favorites", nodes.NewMixFavoritesRankNode)
	
	return registry
}
