package nodes

import (
	"encoding/json"
	"fmt"
	"strings"

	"recommend_engine/internal/model"
	"recommend_engine/internal/workflow"
	"recommend_engine/pkg/llm"
)

type LLMRecallNode struct {
	name      string
	llmClient llm.Client
	promptTpl string
	count     int
}

// NewLLMRecallNode 创建一个新的 LLMRecallNode
// 注意：现在 client 由外部注入，不再负责从 config 创建
func NewLLMRecallNode(name string, client llm.Client, count int) *LLMRecallNode {
	return &LLMRecallNode{
		name:      name,
		llmClient: client,
		count:     count,
	}
}

func (n *LLMRecallNode) Name() string { return n.name }
func (n *LLMRecallNode) Type() string { return "recall" }

func (n *LLMRecallNode) Execute(ctx *workflow.Context) error {
	favorites := ctx.User.Favorites
	if len(favorites) == 0 {
		ctx.AddLog("User has no favorites, skipping LLM recall")
		return nil
	}

	// 构造 Prompt
	// 中文 Prompt 模板
	prompt := fmt.Sprintf(`
用户喜欢以下音乐: %v.
请推荐 %d 首风格相似的、真实存在的、已发行的歌曲。
严禁捏造不存在的歌名，必须是真实歌手演唱的作品。
必须严格输出为 JSON 字符串列表格式，例如 ["歌曲A", "歌曲B"]。
不要包含任何解释、Markdown 格式标记或额外的文本。
确保歌曲名称准确。
`, favorites, n.count)

	messages := []llm.Message{
		{Role: "system", Content: "你是一个专业的音乐推荐引擎。"},
		{Role: "user", Content: prompt},
	}

	// 调用 LLM
	respContent, err := n.llmClient.Chat(ctx.Ctx, messages)
	if err != nil {
		return fmt.Errorf("llm chat failed: %w", err)
	}

	// 尝试清洗和解析 JSON
	cleanedResp := cleanJSON(respContent)
	
	// 解析结果
	var songNames []string
	if err := json.Unmarshal([]byte(cleanedResp), &songNames); err != nil {
		// 记录详细错误日志，包括原始响应
		errMsg := fmt.Sprintf("Failed to parse LLM response: %s. Raw content: [%s]", err, respContent)
		ctx.AddLog(errMsg)
		return fmt.Errorf("failed to parse llm response: %w", err)
	}

	// 转换为 Items
	var items []*model.Item
	for _, name := range songNames {
		// 清理歌名中的书名号
		name = cleanSongName(name)
		if name == "" {
			continue
		}
		
		items = append(items, &model.Item{
			ID:     name, // 简单起见 ID 使用 name
			Name:   name,
			Source: n.name, // 使用节点名作为来源标记 (e.g., llm_gpt4)
		})
	}

	// 写入 Context
	ctx.SetRecallResult(n.name, items)
	ctx.AddLog(fmt.Sprintf("LLM Recall (%s) returned %d items", n.name, len(items)))

	return nil
}

// cleanJSON 尝试从文本中提取并清理 JSON 数组
func cleanJSON(content string) string {
	content = strings.TrimSpace(content)
	
	// 1. 移除 Markdown 代码块标记
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	// 2. 如果包含 '[' 和 ']'，尝试提取中间的部分
	start := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")
	if start != -1 && end != -1 && end > start {
		content = content[start : end+1]
	}

	return content
}

// cleanSongName 去除歌名中的书名号和多余空白
func cleanSongName(name string) string {
	name = strings.ReplaceAll(name, "《", "")
	name = strings.ReplaceAll(name, "》", "")
	return strings.TrimSpace(name)
}
