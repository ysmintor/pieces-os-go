package model

// Role 定义聊天角色类型
type Role string

// 角色常量定义
const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Object 类型定义
type Object string

// Object 类型常量定义
const (
	ObjectChatCompletion      Object = "chat.completion"
	ObjectChatCompletionChunk Object = "chat.completion.chunk"
)

// ChatMessage 聊天消息的基本结构,包含角色和内容
type ChatMessage struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// ChatCompletionRequest 聊天补全API的请求参数结构
type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Stream      bool          `json:"stream"`
	Temperature float64       `json:"temperature"`
	TopP        float64       `json:"top_p"`
}

// ChatCompletionResponse 聊天补全API的响应结构
type ChatCompletionResponse struct {
	ID      string    `json:"id"`
	Object  Object    `json:"object"`
	Created int64     `json:"created"`
	Model   string    `json:"model"`
	Choices []*Choice `json:"choices"`
	Usage   *Usage    `json:"usage"`
}

// FinishReason 类型定义
type FinishReason string

// 预定义的 FinishReason 值
var (
	FinishReasonStop FinishReason = "stop"
)

// Choice 聊天补全响应中的选项内容
type Choice struct {
	Index        int          `json:"index"`
	Message      *ChatMessage `json:"message"`
	FinishReason FinishReason `json:"finish_reason,omitempty"`
}

// Usage 聊天补全响应中的使用情况
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatCompletionStreamResponse 流式聊天补全API的响应结构
type ChatCompletionStreamResponse struct {
	ID      string                        `json:"id"`
	Object  Object                        `json:"object"`
	Created int64                         `json:"created"`
	Model   string                        `json:"model"`
	Choices []*ChatCompletionStreamChoice `json:"choices"`
	Usage   *Usage                        `json:"usage,omitempty"`
}

// ChatCompletionStreamChoice 流式响应中的选项内容
type ChatCompletionStreamChoice struct {
	Index        int                        `json:"index"`
	Delta        *ChatCompletionStreamDelta `json:"delta"`
	FinishReason FinishReason               `json:"finish_reason,omitempty"`
}

// ChatCompletionStreamDelta 流式响应中的增量内容
type ChatCompletionStreamDelta struct {
	Role    Role   `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}
