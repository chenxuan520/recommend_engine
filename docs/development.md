# 开发扩展指南

本指南将帮助你扩展推荐引擎的功能，包括添加新的节点类型和接入新的 LLM 服务。

## 1. 核心概念

*   **Node (节点)**: 工作流中的最小执行单元，实现了 `workflow.Node` 接口。
*   **Registry (注册表)**: 负责将节点类型 (`type`) 映射到具体的构造函数。
*   **Pipeline (流水线)**: 在 `pipelines.json` 中通过配置编排节点的执行顺序。

---

## 2. 如何添加一个新的节点类型

假设我们要添加一个简单的 **"反转排序节点" (ReverseRankNode)** 作为示例。

### 步骤 1: 实现节点逻辑

在 `internal/nodes/` 目录下创建新文件（例如 `rank_reverse.go`），并实现 `workflow.Node` 接口：

```go
package nodes

import (
    "recommend_engine/internal/workflow"
    "recommend_engine/internal/model"
)

// 定义结构体
type ReverseRankNode struct {
    name string
}

// 构造函数
// 注意：必须符合 workflow.NodeFactory 或 setup.go 中闭包的调用方式
func NewReverseRankNode(cfg workflow.NodeConfig) (workflow.Node, error) {
    return &ReverseRankNode{
        name: cfg.Name,
    }, nil
}

// 实现接口方法
func (n *ReverseRankNode) Name() string { return n.name }
func (n *ReverseRankNode) Type() string { return "rank_reverse" }

func (n *ReverseRankNode) Execute(ctx *workflow.Context) error {
    items := ctx.GetCandidates()
    
    // 简单的反转逻辑
    for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
        items[i], items[j] = items[j], items[i]
    }
    
    // 更新上下文
    ctx.UpdateCandidates(items)
    ctx.AddLog("Candidates reversed")
    return nil
}
```

### 步骤 2: 注册节点

打开 `cmd/recommend/setup.go`，在 `RegisterNodes` 函数中注册这个新节点：

```go
func RegisterNodes(...) *workflow.Registry {
    // ... 其他节点注册 ...

    // 注册新节点
    // 这里的字符串 "rank_reverse" 就是 configs/pipelines.json 中使用的 "type"
    registry.Register("rank_reverse", func(cfg workflow.NodeConfig) (workflow.Node, error) {
        return nodes.NewReverseRankNode(cfg)
    })
    
    return registry
}
```

### 步骤 3: 配置使用

在 `configs/pipelines.json` 的 `nodes` 列表中添加配置：

```json
{
  "name": "my_reverse_rank",
  "type": "rank_reverse",
  "config": {}
}
```

---

## 3. 如何添加一个新的 LLM 召回节点

通常情况下，你不需要写代码，只需要修改配置。

### 场景 A: 接入兼容 OpenAI 接口的模型 (推荐)

目前的 `recall_llm` 节点是通用的，支持任何兼容 OpenAI API 格式的服务（如 DeepSeek, Moonshot, 通义千问等）。

**1. 修改 `configs/llm.yaml`**

添加新的模型服务商配置：

```yaml
llms:
  xinhuo:
    # ... (原有配置)
  
  # 新增 DeepSeek 配置
  deepseek:
    chat_endpoint: "https://api.deepseek.com/v1/chat/completions"
    api_key: "sk-your-deepseek-key"
    model: "deepseek-chat"
```

**2. 修改 `configs/pipelines.json`**

在 `nodes` 列表中（通常在 `parallel` 组里）添加一个使用新配置的节点：

```json
{
  "name": "deepseek_recall_node",
  "type": "recall_llm",
  "config": {
    "llm_config_key": "deepseek",  // <--- 对应 llm.yaml 中的 key
    "count": 20
  }
}
```

### 场景 B: 接入不兼容 OpenAI 接口的模型

如果目标 LLM 的 API 格式完全不同，你需要编写适配代码。

1.  **定义接口**: 在 `pkg/llm/client.go` 中查看 `Client` 接口定义。
2.  **实现 Client**: 在 `pkg/llm/` 下新建文件（如 `gemini.go`），实现该接口。
3.  **修改注册逻辑**: 
    *   修改 `cmd/recommend/setup.go`。
    *   在注册 `recall_llm` 时，检查 `llmCfg` 中的类型字段（可能需要扩展 `LLMGlobalConfig` 结构体来支持 distinguishing provider type）。
    *   根据类型初始化不同的 Client (OpenAIClient vs GeminiClient)。
