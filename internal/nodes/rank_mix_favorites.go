package nodes

import (
	"fmt"
	"math/rand"
	"time"

	"recommend_engine/internal/model"
	"recommend_engine/internal/workflow"
)

type MixFavoritesRankNode struct {
	name     string
	mixCount int
}

func NewMixFavoritesRankNode(cfg workflow.NodeConfig) (workflow.Node, error) {
	count, _ := cfg.Config["mix_count"].(float64)
	if count <= 0 {
		count = 2 // 默认插入 2 首
	}

	return &MixFavoritesRankNode{
		name:     cfg.Name,
		mixCount: int(count),
	}, nil
}

func (n *MixFavoritesRankNode) Name() string { return n.name }
func (n *MixFavoritesRankNode) Type() string { return "rank" }

func (n *MixFavoritesRankNode) Execute(ctx *workflow.Context) error {
	favorites := ctx.User.Favorites
	if len(favorites) == 0 {
		return nil
	}

	// 1. 随机选取 Favorites
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	shuffledFavs := make([]string, len(favorites))
	copy(shuffledFavs, favorites)
	
	r.Shuffle(len(shuffledFavs), func(i, j int) {
		shuffledFavs[i], shuffledFavs[j] = shuffledFavs[j], shuffledFavs[i]
	})

	pickCount := n.mixCount
	if pickCount > len(shuffledFavs) {
		pickCount = len(shuffledFavs)
	}
	picks := shuffledFavs[:pickCount]

	// 2. 转换为 Items
	var mixItems []*model.Item
	for _, name := range picks {
		mixItems = append(mixItems, &model.Item{
			ID:     name,
			Name:   name,
			Score:  100.0, // 给予高分，或者仅仅作为特殊标记
			Source: "user_favorite",
			MetaData: map[string]interface{}{
				"is_mix_in": true,
			},
		})
	}

	// 3. 混合策略
	// 策略：随机插入到 Candidates 中
	candidates := ctx.GetCandidates()
	
	// 合并
	finalList := append(candidates, mixItems...)
	
	// 再次打乱整个列表，或者保持 mixItems 在前/后?
	// 需求是"随机选择插入"，通常意味着位置随机。
	// 但为了保持原有排序逻辑（如果有），可能需要在之后再做一次 shuffle，或者这里手动 insert。
	// 简单起见，我们把它们加进去，然后整体做一次 Shuffle (如果这是最后一步)。
	// 但如果前面已经排好序了，我们可能希望随机替换掉某些位置，或者随机插入。
	
	// 这里采用“随机插入”策略：
	// 创建一个足够大的 slice，先把原有的填进去，再把 mix 的插进去。
	// 或者简单粗暴：Append 后 Shuffle。
	r.Shuffle(len(finalList), func(i, j int) {
		finalList[i], finalList[j] = finalList[j], finalList[i]
	})

	ctx.UpdateCandidates(finalList)
	ctx.AddLog(fmt.Sprintf("MixFavorites (%s) injected %d items", n.name, len(mixItems)))

	return nil
}
