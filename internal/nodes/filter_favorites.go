package nodes

import (
	"fmt"

	"recommend_engine/internal/model"
	"recommend_engine/internal/workflow"
)

type FavoritesFilterNode struct {
	name string
}

func NewFavoritesFilterNode(cfg workflow.NodeConfig) (workflow.Node, error) {
	return &FavoritesFilterNode{
		name: cfg.Name,
	}, nil
}

func (n *FavoritesFilterNode) Name() string { return n.name }
func (n *FavoritesFilterNode) Type() string { return "filter" }

func (n *FavoritesFilterNode) Execute(ctx *workflow.Context) error {
	candidates := ctx.GetCandidates()
	if len(candidates) == 0 {
		return nil
	}

	// 构建收藏 Set
	favSet := make(map[string]struct{})
	for _, name := range ctx.User.Favorites {
		favSet[name] = struct{}{}
	}

	var kept []*model.Item
	filteredCount := 0

	for _, item := range candidates {
		if _, exists := favSet[item.Name]; !exists {
			kept = append(kept, item)
		} else {
			filteredCount++
		}
	}

	ctx.UpdateCandidates(kept)
	ctx.AddLog(fmt.Sprintf("Favorites filter (%s) removed %d items, kept %d", n.name, filteredCount, len(kept)))
	return nil
}
