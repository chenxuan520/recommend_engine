package workflow

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

// ParallelNode 是一个组合节点，用于并发执行多个子节点
type ParallelNode struct {
	nodeName string
	children []Node
}

// NewParallelNode 创建一个新的并行节点
func NewParallelNode(name string, children []Node) *ParallelNode {
	return &ParallelNode{
		nodeName: name,
		children: children,
	}
}

func (n *ParallelNode) Name() string {
	return n.nodeName
}

func (n *ParallelNode) Type() string {
	return "parallel"
}

// Execute 并发执行所有子节点
// 采用 "Best Effort" 策略：只要有一个子节点成功，就不视为整个节点失败。
// 只有当所有子节点都失败时，才返回错误。
func (n *ParallelNode) Execute(ctx *Context) error {
	ctx.AddLog(fmt.Sprintf("Start ParallelNode: %s", n.nodeName))

	var wg sync.WaitGroup
	var successCount int32
	var errors []string
	var mu sync.Mutex // 保护 errors 切片

	for _, child := range n.children {
		wg.Add(1)
		go func(node Node) {
			defer wg.Done()
			
			// 可以在这里增加 recover 防止 panic 导致 crash
			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					errors = append(errors, fmt.Sprintf("node %s panic: %v", node.Name(), r))
					mu.Unlock()
				}
			}()

			ctx.AddLog(fmt.Sprintf("  -> Start child node: %s", node.Name()))
			if err := node.Execute(ctx); err != nil {
				ctx.AddLog(fmt.Sprintf("  -> Node %s failed: %v", node.Name(), err))
				mu.Lock()
				errors = append(errors, fmt.Sprintf("node %s: %v", node.Name(), err))
				mu.Unlock()
			} else {
				atomic.AddInt32(&successCount, 1)
				ctx.AddLog(fmt.Sprintf("  -> Node %s completed", node.Name()))
			}
		}(child)
	}

	wg.Wait()

	// 决策逻辑：
	// 1. 如果有至少一个成功，则认为整体成功（Partial Success）
	// 2. 如果所有都失败，则返回聚合错误
	if successCount == 0 && len(errors) > 0 {
		return fmt.Errorf("all parallel nodes failed: %s", strings.Join(errors, "; "))
	}

	if len(errors) > 0 {
		ctx.AddLog(fmt.Sprintf("ParallelNode completed with %d errors (ignored due to partial success): %v", len(errors), errors))
	} else {
		ctx.AddLog(fmt.Sprintf("End ParallelNode: %s (All success)", n.nodeName))
	}
	
	return nil
}
