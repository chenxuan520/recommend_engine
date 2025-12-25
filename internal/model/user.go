package model

// User 代表系统中的用户信息
type User struct {
	ID        string   `json:"id" yaml:"id"`
	Token     string   `json:"-" yaml:"token"` // Token 用于鉴权，不序列化到 JSON
	Name      string   `json:"name" yaml:"name"`
	Favorites []string `json:"favorites" yaml:"favorites"` // 用户的收藏列表，用于构建召回 Prompt
}
