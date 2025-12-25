package workflow

import (
	"context"
	"sync"

	"recommend_engine/internal/model"
)

// Context 承载推荐流程的所有状态信息
// 它是并发安全的，支持多路召回并行写入
type Context struct {
	Ctx    context.Context
	UserID string
	User   *model.User
	Config map[string]interface{}

	// 数据流转区 (需要锁保护)
	mu            sync.RWMutex
	Candidates    []*model.Item             // 当前的主候选集
	RecallResults map[string][]*model.Item  // 各路召回的原始结果 key: source_name
	TraceLog      []string                  // 执行日志
}

// NewContext 创建一个新的工作流上下文
func NewContext(ctx context.Context, userID string, user *model.User) *Context {
	return &Context{
		Ctx:           ctx,
		UserID:        userID,
		User:          user,
		RecallResults: make(map[string][]*model.Item),
		Candidates:    make([]*model.Item, 0),
		TraceLog:      make([]string, 0),
	}
}

// AddCandidates 向候选集中添加项目 (线程安全)
func (c *Context) AddCandidates(items []*model.Item) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Candidates = append(c.Candidates, items...)
}

// SetRecallResult 记录特定召回源的结果 (线程安全)
func (c *Context) SetRecallResult(source string, items []*model.Item) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.RecallResults[source] = items
	// 通常召回结果也会直接合并到 Candidates 中
	c.Candidates = append(c.Candidates, items...)
}

// GetCandidates 获取当前候选集的副本 (线程安全)
func (c *Context) GetCandidates() []*model.Item {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// 返回副本以防止并发读写问题，或者由调用方保证后续只读
	// 这里简单返回切片副本
	result := make([]*model.Item, len(c.Candidates))
	copy(result, c.Candidates)
	return result
}

// UpdateCandidates 更新整个候选集 (线程安全)
// 通常用于过滤或排序阶段
func (c *Context) UpdateCandidates(items []*model.Item) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Candidates = items
}

// AddLog 添加追踪日志
func (c *Context) AddLog(msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.TraceLog = append(c.TraceLog, msg)
}

// Node 定义工作流中的执行节点
type Node interface {
	Name() string
	Type() string // e.g., "recall", "filter", "rank", "parallel"
	Execute(ctx *Context) error
}
