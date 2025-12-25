# 架构设计文档

## 概述

本系统旨在构建一个高度可扩展、配置驱动的推荐引擎框架，MVP 版本专注于利用大语言模型（LLM）进行音乐推荐。系统核心采用**Pipeline/Workflow** 模式，将推荐流程拆解为一系列可独立执行、可编排的 **节点 (Node)**。通过 JSON 配置文件定义 Node 的执行顺序，实现流程的动态编排。

## 架构

系统采用 **配置驱动的管道架构 (Configuration-Driven Pipeline Architecture)**：

1.  **流程上下文 (Workflow Context)**: 一个贯穿整个请求生命周期的“大结构体”，承载所有状态（用户信息、候选集、中间结果、最终结果）。
2.  **节点 (Node)**: 最小执行单元。每个 Node 负责处理 Context 中的特定数据。
    *   **Recall Node**: 负责召回（如 LLM 召回）。
    *   **Filter Node**: 负责过滤（如历史去重）。
    *   **Rank Node**: 负责排序（粗排/精排）。
    *   **ReRank Node**: 负责重排（混排）。
3.  **管道引擎 (Pipeline Engine)**: 负责解析 JSON 编排配置，构建执行链路，并依次驱动 Node 执行。
4.  **节点注册表 (Node Registry)**: 负责注册和管理所有可用的 Node 实现，支持插件化扩展。

```mermaid
graph TD
    Config[Pipeline Config (JSON)] --> Engine[Pipeline Engine]
    Registry[Node Registry] -.-> Engine
    
    subgraph "Execution Flow"
        Ctx[Workflow Context]
        Engine --Execute Node 1--> Node1[LLM Recall Node]
        Node1 --Update Ctx--> Ctx
        Engine --Execute Node 2--> Node2[History Filter Node]
        Node2 --Update Ctx--> Ctx
        Engine --Execute Node 3--> Node3[Simple Rank Node]
        Node3 --Update Ctx--> Ctx
    end
    
    User[User Provider] --> Ctx
    DB[History Store] --> Node2
    LLM[LLM Service] --> Node1
```

## 组件和接口

### 1. Workflow Core (`internal/workflow`)

*   **`Context`**: 核心上下文结构体，在 Node 间传递。**注意：必须保证并发安全，特别是写操作。**
    ```go
    type WorkflowContext struct {
        Ctx          context.Context
        UserID       string
        User         *User              
        Config       map[string]interface{} 
        
        // 数据流转区 (并发安全访问)
        mu             sync.RWMutex
        Candidates     []*Item            // 当前候选集（随流程不断变化）
        RecallResults  map[string][]*Item // Key: 召回源名称
        
        // ...
    }
    ```

*   **`Node`**: 通用节点接口。
*   **`ParallelNode`**: 组合节点，负责并发执行子节点。
    *   类型标识: `group` 或 `parallel`
    *   逻辑: 启动多个 Goroutine 执行子节点，等待所有完成，并处理错误聚合。

*   **`Engine`**: 流程执行引擎。
    *   `NewEngine(config PipelinesConfig, registry *Registry) *Engine`
    *   `Run(ctx *WorkflowContext, scene string) error` // 根据 scene 选择执行哪条 pipeline

### 2. Pipeline Configuration

*   **`pipelines.json`**: 编排配置文件。支持为每个场景配置独立的属性和流程。
    *   引入 `type: "parallel"` 节点支持多路并发召回。
    ```json
    {
      "pipelines": {
        "music": {
          "description": "主音乐推荐流",
          "timeout_ms": 5000,
          "nodes": [
            {
              "name": "multi_source_recall",
              "type": "parallel",
              "nodes": [
                {
                  "name": "llm_gpt4_recall",
                  "type": "recall",
                  "config": { "model": "gpt-4" }
                },
                {
                  "name": "llm_deepseek_recall",
                  "type": "recall",
                  "config": { "model": "deepseek-v3" }
                }
              ]
            },
            {
              "name": "history_dedup_filter",
              "type": "filter",
              "config": {
                "lookback_days": 7
              }
            },
            {
              "name": "simple_score_rank",
              "type": "rank",
              "config": {
                "order": "desc"
              }
            }
          ]
        }
      }
    }
    ```

### 3. Node Implementations (`internal/nodes`)

*   **`LLMRecallNode`**: 实现 `Node` 接口。
    *   从 `WorkflowContext` 读取 `User.Favorites`。
    *   调用 LLM 服务。
    *   将结果写入 `WorkflowContext.Candidates`。
*   **`HistoryFilterNode`**: 实现 `Node` 接口。
    *   读取 `WorkflowContext.Candidates`。
    *   调用 `HistoryStore` 获取历史。
    *   剔除重复项，更新 `WorkflowContext.Candidates`。
*   **`RankNode`**: 实现 `Node` 接口。
    *   对 `WorkflowContext.Candidates` 进行排序。

## 数据模型

### User
```go
type User struct {
    ID        string
    Name      string
    Favorites []string 
}
```

### Item
```go
type Item struct {
    ID       string
    Name     string
    Score    float64
    Source   string // 召回源标记
    MetaData map[string]interface{}
}
```

## 错误处理

*   **Node 执行失败**: 
    *   **Fail-Fast**: 关键节点（如所有召回都失败）直接返回错误。
    *   **Fail-Safe**: 非关键节点（如某个辅助排序失效）记录日志并跳过，流程继续。
    *   可在 JSON 配置中为每个 Node 指定 `on_error: "continue" | "abort"`。

## 测试策略

1.  **单元测试**:
    *   测试各个 `Node` 的 `Execute` 方法，Mock `WorkflowContext` 数据。
2.  **引擎测试**:
    *   构建一个包含 Mock Node 的 Pipeline，验证执行顺序和 Context 传递是否正确。
3.  **集成测试**:
    *   加载真实的 `pipeline.json`，运行完整流程。
