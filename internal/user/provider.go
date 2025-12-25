package user

import (
	"fmt"
	"os"
	"sync"

	"recommend_engine/internal/model"

	"gopkg.in/yaml.v3"
)

// Provider 定义了用户数据获取的接口
type Provider interface {
	GetUser(userID string) (*model.User, error)
	GetUserByToken(token string) (*model.User, error) // 新增接口
}

// StaticProvider 基于静态配置文件实现的用户提供者
type StaticProvider struct {
	users      map[string]*model.User
	tokenIndex map[string]*model.User // 新增 Token 索引
	mu         sync.RWMutex
}

type staticConfig struct {
	Users []model.User `yaml:"users"`
}

// NewStaticProvider 创建一个新的 StaticProvider 实例
// configPath 是用户配置文件的路径 (yaml格式)
func NewStaticProvider(configPath string) (*StaticProvider, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read user config file: %w", err)
	}

	var config staticConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse user config: %w", err)
	}

	userMap := make(map[string]*model.User)
	tokenIndex := make(map[string]*model.User)

	for i := range config.Users {
		u := config.Users[i]
		// 注意：这里需要深拷贝或取地址，确保 map 指向正确的数据
		// 由于 u 是循环变量，在 Go 1.22 之前如果直接取地址会有问题，但 config.Users[i] 安全
		userPtr := &config.Users[i]
		userMap[u.ID] = userPtr
		
		if u.Token != "" {
			tokenIndex[u.Token] = userPtr
		}
	}

	return &StaticProvider{
		users:      userMap,
		tokenIndex: tokenIndex,
	}, nil
}

// GetUser 根据 UserID 获取用户信息
func (p *StaticProvider) GetUser(userID string) (*model.User, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	u, ok := p.users[userID]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", userID)
	}
	return u, nil
}

// GetUserByToken 根据 Token 获取用户信息
func (p *StaticProvider) GetUserByToken(token string) (*model.User, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	u, ok := p.tokenIndex[token]
	if !ok {
		return nil, fmt.Errorf("invalid token")
	}
	return u, nil
}
