# LLM Recommendation Engine

这是一个基于大语言模型（LLM）的通用推荐引擎框架，目前实现了音乐推荐场景。系统采用 **Pipeline** 架构，支持多路并发召回、历史去重、收藏过滤和随机混排。

## 功能特性

*   **多路并发召回**: 支持配置多个 LLM 节点并发执行，提高响应速度和多样性。
*   **Pipeline 编排**: 通过 `pipelines.json` 灵活定义推荐流程（召回 -> 过滤 -> 排序 -> 混排）。
*   **动态上下文**: 支持通过 HTTP POST 请求动态传入用户收藏列表 (`favorites`) 作为推荐种子。
*   **智能过滤**:
    *   **历史去重**: 自动记录推荐历史，避免 7 天内重复推荐。
    *   **收藏过滤**: 自动过滤用户已收藏的歌曲。
*   **多样性策略**: 支持随机“回捞”少量用户收藏歌曲混入推荐列表，增加亲切感。
*   **中文优化**: 针对中文歌曲进行了 Prompt 和解析清洗优化，去除书名号。

## 快速开始

### 1. 配置

确保 `configs/` 目录下有以下配置文件：

*   `configs/users.yaml`: 定义用户和 Token（用于鉴权）。
*   `configs/llm.yaml`: 定义 LLM 的 Endpoint 和 API Key。
*   `configs/pipelines.json`: 定义推荐流程。

### 2. 运行

```bash
go run cmd/recommend/main.go
```

服务默认启动在 `:8080` 端口。

### 3. 测试

使用 `test/run.sh` 脚本进行自动化集成测试：

```bash
./test/run.sh
```

或者手动发送请求：

```bash
curl -X POST -H "Authorization: Bearer sk-token-alice" \
     -H "Content-Type: application/json" \
     -d '{
           "scene": "music",
           "favorites": ["写给黄淮", "可能否", "童话镇"]
         }' \
     "http://localhost:8080/api/v1/recommend"
```

## 目录结构

*   `cmd/`: 程序入口。
*   `internal/`: 核心业务逻辑。
    *   `workflow/`: Pipeline 引擎核心。
    *   `nodes/`: 具体的业务节点实现 (LLMRecall, Filter, Rank)。
    *   `server/`: HTTP Server 实现。
*   `pkg/`: 通用工具库 (LLM Client)。
*   `configs/`: 配置文件。
*   `test/`: 测试脚本和数据。

