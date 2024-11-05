package tokenizer

import (
	"fmt"
	"os"
	"pieces-os-go/internal/model"
	"pieces-os-go/pkg/tiktoken_loader"
	"pieces-os-go/pkg/tiktoken_loader/assets"
	"strings"

	"github.com/pkoukk/tiktoken-go"
	"github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/pretrained"
)

var (
	tiktokenCl100k  *tiktoken.Tiktoken
	tiktokenO200k   *tiktoken.Tiktoken
	claudeTokenizer *tokenizer.Tokenizer
)

// InitTokenizers 初始化所有tokenizer
func InitTokenizers() error {
	// 初始化tiktoken
	tiktoken.SetBpeLoader(tiktoken_loader.NewOfflineLoader())
	var err error
	tiktokenCl100k, err = tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return err
	}
	tiktokenO200k, err = tiktoken.GetEncoding("o200k_base")
	if err != nil {
		return err
	}

	// 初始化Claude tokenizer
	tokenizerJSON, err := assets.Assets.ReadFile("tokenizer.json")
	if err != nil {
		return err
	}
	claudeTokenizer, err = NewTokenizer(tokenizerJSON)
	if err != nil {
		return err
	}

	return nil
}

// getModelTokens 根据模型获取 tokensPerMessage 和 tokensPerName
func getModelTokens(model string) (tokensPerMessage int, tokensPerName int) {
	switch model {
	case "gpt-3.5-turbo-0613",
		"gpt-3.5-turbo-16k-0613",
		"gpt-4-0314",
		"gpt-4-32k-0314",
		"gpt-4-0613",
		"gpt-4-32k-0613",
		"gpt-4o-mini-2024-07-18",
		"gpt-4o-2024-08-06":
		return 3, 1
	case "gpt-3.5-turbo-0301":
		return 4, -1
	default:
		if strings.Contains(model, "gpt-3.5-turbo") {
			return getModelTokens("gpt-3.5-turbo-0613")
		} else if strings.Contains(model, "gpt-4o-mini") {
			return getModelTokens("gpt-4o-mini-2024-07-18")
		} else if strings.Contains(model, "gpt-4o") {
			return getModelTokens("gpt-4o-2024-08-06")
		} else if strings.Contains(model, "gpt-4") {
			return getModelTokens("gpt-4-0613")
		} else {
			return -1, -1
		}
	}
}

// tokensFromMessage 计算单个消息的Tokens数量
func tokensFromMessage(message *model.ChatMessage, tokensPerMessage, tokensPerName int, tiktoken *tiktoken.Tiktoken) int {
	numTokens := tokensPerMessage
	numTokens += len(tiktoken.Encode(message.Content, nil, nil))
	numTokens += len(tiktoken.Encode(string(message.Role), nil, nil))
	if message.Name != "" {
		numTokens += tokensPerName
		numTokens += len(tiktoken.Encode(message.Name, nil, nil))
	}
	return numTokens
}

// getTiktoken 根据模型名返回对应的tiktoken实例
func getTiktoken(model string) *tiktoken.Tiktoken {
	encoding := getEncoding(model)
	switch encoding {
	case "o200k_base":
		return tiktokenO200k
	case "cl100k_base":
		return tiktokenCl100k
	default:
		return tiktokenCl100k // 默认使用cl100k_base编码
	}
}

// NumTokensFromText 计算字符串的Tokens数量
func NumTokensFromText(text string, model string) int {
	tiktoken := getTiktoken(model)
	return len(tiktoken.Encode(text, nil, nil))
}

// NumTokensFromMessage 计算单条消息的Tokens数量
func NumTokensFromMessage(message *model.ChatMessage, model string) int {
	tokensPerMessage, tokensPerName := getModelTokens(model)
	if tokensPerMessage == -1 {
		tokensPerMessage, tokensPerName = 3, 1
	}
	return tokensFromMessage(message, tokensPerMessage, tokensPerName, getTiktoken(model))
}

// OpenAI Cookbook: https://raw.githubusercontent.com/openai/openai-cookbook/refs/heads/main/examples/How_to_count_tokens_with_tiktoken.ipynb
// NumTokensFromMessages 计算多条消息的Tokens总数量
func NumTokensFromMessages(messages []model.ChatMessage, model string) (numTokens int) {
	tokensPerMessage, tokensPerName := getModelTokens(model)
	if tokensPerMessage == -1 {
		tokensPerMessage, tokensPerName = 3, 1
	}
	for _, message := range messages {
		numTokens += tokensFromMessage(&message, tokensPerMessage, tokensPerName, getTiktoken(model))
	}
	numTokens += 3
	return numTokens
}

// NewTokenizer 从JSON数据创建新的tokenizer
func NewTokenizer(tokenizerJSON []byte) (*tokenizer.Tokenizer, error) {
	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "tokenizer-*.json")
	if err != nil {
		return nil, fmt.Errorf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// 将JSON数据写入临时文件
	if _, err := tmpFile.Write(tokenizerJSON); err != nil {
		return nil, fmt.Errorf("写入临时文件失败: %v", err)
	}

	// 使用临时文件初始化tokenizer
	tk, err := pretrained.FromFile(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("初始化tokenizer失败: %v", err)
	}

	return tk, nil
}

// CountTokens 计算文本中的token数量
func CountTokens(text string) (int, error) {
	if claudeTokenizer == nil {
		return 0, ErrTokenizerNotInitialized
	}
	encoding, err := claudeTokenizer.EncodeSingle(text)
	if err != nil {
		return 0, fmt.Errorf("计算token数量失败: %v", err)
	}
	return encoding.Len(), nil
}
