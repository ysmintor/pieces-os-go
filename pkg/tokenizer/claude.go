package tokenizer

import (
	"pieces-os-go/internal/model"
	"strings"
)

// CountClaudeTokens 计算文本的token数量
func CountClaudeTokens(text string) (int, error) {
	if claudeTokenizer == nil {
		return 0, ErrTokenizerNotInitialized
	}
	encoding, err := claudeTokenizer.EncodeSingle(text)
	if err != nil {
		return 0, err
	}
	return encoding.Len(), nil
}

// NumTokensFromClaudeMessage 计算Claude消息的token数量
func NumTokensFromClaudeMessage(message *model.ChatMessage) (int, error) {
	// 根据Claude的实现，我们只需要计算 role + ": " + content
	text := string(message.Role) + ": " + message.Content
	return CountTokens(text)
}

// NumTokensFromClaudeMessages 计算多条Claude消息的总token数量
func NumTokensFromClaudeMessages(params *TokenCountParams) (int, error) {
	var prompt strings.Builder

	// 添加system prompt (如果有)
	if params.System != "" {
		prompt.WriteString("\n\nSystem:")
		prompt.WriteString(params.System)
	}

	// 按Claude格式拼接消息
	for _, msg := range params.Messages {
		prompt.WriteString("\n\n")
		prompt.WriteString(string(msg.Role))
		prompt.WriteString(":")
		prompt.WriteString(msg.Content)
	}

	// 如果最后一条消息是用户消息，添加Assistant:前缀
	if len(params.Messages) > 0 && params.Messages[len(params.Messages)-1].Role == "user" {
		prompt.WriteString("\n\nAssistant:")
	}

	return CountTokens(prompt.String())
}

// TokenCountParams 定义计算tokens所需的参数
type TokenCountParams struct {
	Messages []model.ChatMessage
	System   string
}
