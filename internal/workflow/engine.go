package workflow

import (
	"encoding/json"
	"fmt"
	"os"
)

// PipelineConfig 单个 Pipeline 的配置
type PipelineConfig struct {
	Description string       `json:"description"`
	TimeoutMs   int          `json:"timeout_ms"`
	Nodes       []NodeConfig `json:"nodes"`
}

// NodeConfig 节点的配置片段
type NodeConfig struct {
	Name   string                 `json:"name"`
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
	Nodes  []NodeConfig           `json:"nodes,omitempty"` // 用于组合节点 (如 parallel)
}

// GlobalConfig 整个配置文件的结构
type GlobalConfig struct {
	Pipelines map[string]PipelineConfig `json:"pipelines"`
}

// NodeFactory 创建 Node 的函数签名
type NodeFactory func(config NodeConfig) (Node, error)

// Registry 节点注册表
type Registry struct {
	factories map[string]NodeFactory
}

func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]NodeFactory),
	}
}

// Register 注册一个新的节点类型
func (r *Registry) Register(nodeType string, factory NodeFactory) {
	r.factories[nodeType] = factory
}

// CreateNode 根据配置创建节点实例
func (r *Registry) CreateNode(cfg NodeConfig) (Node, error) {
	// 特殊处理 parallel 节点，因为它属于框架层面的能力
	if cfg.Type == "parallel" {
		var children []Node
		for _, childCfg := range cfg.Nodes {
			childNode, err := r.CreateNode(childCfg)
			if err != nil {
				return nil, err
			}
			children = append(children, childNode)
		}
		return NewParallelNode(cfg.Name, children), nil
	}

	factory, ok := r.factories[cfg.Type]
	if !ok {
		return nil, fmt.Errorf("unknown node type: %s", cfg.Type)
	}
	return factory(cfg)
}

// Engine 流程引擎
type Engine struct {
	pipelines map[string][]Node // scene -> nodes
	registry  *Registry
}

// NewEngine 创建引擎并加载配置
func NewEngine(configPath string, registry *Registry) (*Engine, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pipeline config: %w", err)
	}

	var globalCfg GlobalConfig
	if err := json.Unmarshal(data, &globalCfg); err != nil {
		return nil, fmt.Errorf("failed to parse pipeline config: %w", err)
	}

	engine := &Engine{
		pipelines: make(map[string][]Node),
		registry:  registry,
	}

	for scene, pipeCfg := range globalCfg.Pipelines {
		var nodes []Node
		for _, nodeCfg := range pipeCfg.Nodes {
			node, err := registry.CreateNode(nodeCfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create node '%s' in pipeline '%s': %w", nodeCfg.Name, scene, err)
			}
			nodes = append(nodes, node)
		}
		engine.pipelines[scene] = nodes
	}

	return engine, nil
}

// Run 执行指定场景的推荐流程
func (e *Engine) Run(ctx *Context, scene string) error {
	nodes, ok := e.pipelines[scene]
	if !ok {
		return fmt.Errorf("pipeline not found for scene: %s", scene)
	}

	ctx.AddLog(fmt.Sprintf("Starting pipeline execution for scene: %s", scene))

	for _, node := range nodes {
		ctx.AddLog(fmt.Sprintf("Executing node: %s (%s)", node.Name(), node.Type()))
		if err := node.Execute(ctx); err != nil {
			ctx.AddLog(fmt.Sprintf("Node execution failed: %v", err))
			return err
		}
	}

	ctx.AddLog("Pipeline execution completed")
	return nil
}
