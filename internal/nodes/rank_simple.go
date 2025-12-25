package nodes

import (
	"fmt"
	"math/rand"
	"sort"
	"time"

	"recommend_engine/internal/workflow"
)

type SimpleRankNode struct {
	name  string
	limit int
	order string // "desc", "asc", "shuffle"
}

func NewSimpleRankNode(cfg workflow.NodeConfig) (workflow.Node, error) {
	limit, _ := cfg.Config["limit"].(float64)
	order, _ := cfg.Config["order"].(string)

	if order == "" {
		order = "shuffle" // 默认打乱，因为 MVP 中 LLM 返回的顺序可能就是相关性顺序，但也可能需要打散
	}

	return &SimpleRankNode{
		name:  cfg.Name,
		limit: int(limit),
		order: order,
	}, nil
}

func (n *SimpleRankNode) Name() string { return n.name }
func (n *SimpleRankNode) Type() string { return "rank" }

func (n *SimpleRankNode) Execute(ctx *workflow.Context) error {
	candidates := ctx.GetCandidates()
	if len(candidates) == 0 {
		return nil
	}

	// 排序逻辑
	switch n.order {
	case "desc":
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].Score > candidates[j].Score
		})
	case "asc":
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].Score < candidates[j].Score
		})
	case "shuffle":
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		r.Shuffle(len(candidates), func(i, j int) {
			candidates[i], candidates[j] = candidates[j], candidates[i]
		})
	}

	// 截断
	if n.limit > 0 && len(candidates) > n.limit {
		candidates = candidates[:n.limit]
	}

	ctx.UpdateCandidates(candidates)
	ctx.AddLog(fmt.Sprintf("Rank (%s) completed. Strategy: %s, Result count: %d", n.name, n.order, len(candidates)))

	return nil
}
