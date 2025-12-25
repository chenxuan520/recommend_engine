package model

// Item 代表推荐系统中的一个条目（如一首歌、一部电影）
type Item struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Score    float64                `json:"score"`              // 排序分数
	Source   string                 `json:"source"`             // 召回源标记 (e.g., "llm_gpt4", "hot_list")
	MetaData map[string]interface{} `json:"meta_data,omitempty"` // 额外的元数据
}
