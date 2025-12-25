package nodes

import (
	"fmt"

	"recommend_engine/internal/history"
	"recommend_engine/internal/model"
	"recommend_engine/internal/workflow"
)

type HistoryFilterNode struct {
	name         string
	store        history.Store
	lookbackDays int
}

// NewHistoryFilterNode 工厂函数
func NewHistoryFilterNode(cfg workflow.NodeConfig, store history.Store) (workflow.Node, error) {
	days, ok := cfg.Config["lookback_days"].(float64)
	if !ok {
		days = 7 // default
	}

	return &HistoryFilterNode{
		name:         cfg.Name,
		store:        store,
		lookbackDays: int(days),
	}, nil
}

func (n *HistoryFilterNode) Name() string { return n.name }
func (n *HistoryFilterNode) Type() string { return "filter" }

func (n *HistoryFilterNode) Execute(ctx *workflow.Context) error {
	candidates := ctx.GetCandidates()
	if len(candidates) == 0 {
		return nil
	}

	// 获取历史记录
	// 注意：这里需要知道当前的 Domain。
	// 简单起见，假设 Config 或者 Scene 名字就是 Domain，或者在 NodeConfig 里配置 domain。
	// 这里我们尝试从 Context.Config 获取，或者默认 "music"
	domain := "music" // Default
	if d, ok := ctx.Config["domain"].(string); ok {
		domain = d
	}

	historyItems, err := n.store.GetRecentHistory(ctx.UserID, domain, n.lookbackDays)
	if err != nil {
		// 历史获取失败是否阻断流程？
		// 策略：记录日志，降级为不顾虑历史
		ctx.AddLog(fmt.Sprintf("Failed to get history: %v", err))
		return nil 
	}

	// 构建历史 Set
	historySet := make(map[string]struct{})
	for _, item := range historyItems {
		historySet[item] = struct{}{}
	}

	// 过滤
	var kept []*model.Item
	filteredCount := 0
	for _, item := range candidates {
		if _, exists := historySet[item.Name]; !exists {
			kept = append(kept, item)
		} else {
			filteredCount++
		}
	}

	// 更新 Context
	ctx.UpdateCandidates(kept)
	ctx.AddLog(fmt.Sprintf("History filter (%s) removed %d items, kept %d", n.name, filteredCount, len(kept)))

	return nil
}
