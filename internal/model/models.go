package model

import (
	"encoding/json"
	"fmt"
	"pieces-os-go/assets"
	"strings"
	"sync"
	"time"
)

type Model struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	OwnedBy string                 `json:"owned_by"`
	Details map[string]interface{} `json:"details,omitempty"`
}

type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

var (
	SupportedModels map[string]Model
	initModelsOnce  sync.Once
)

// var SupportedModels = map[string]Model{
// 	"chat-bison": {
// 		ID:      "chat-bison",
// 		Object:  "model",
// 		Created: 1694793600,
// 		OwnedBy: "google",
// 	},
// 	"gpt-4o-mini": {
// 		ID:      "gpt-4o-mini",
// 		Object:  "model",
// 		Created: 1721232000,
// 		OwnedBy: "openai",
// 	},
// 	"gemini-1.5-pro": {
// 		ID:      "gemini-1.5-pro",
// 		Object:  "model",
// 		Created: 1716825600,
// 		OwnedBy: "google",
// 	},
// 	"gpt-4o": {
// 		ID:      "gpt-4o",
// 		Object:  "model",
// 		Created: 1715702400,
// 		OwnedBy: "openai",
// 	},
// 	"codechat-bison": {
// 		ID:      "codechat-bison",
// 		Object:  "model",
// 		Created: 1694793600,
// 		OwnedBy: "google",
// 	},
// 	"claude-3-sonnet@20240229": {
// 		ID:      "claude-3-sonnet@20240229",
// 		Object:  "model",
// 		Created: 1709136000,
// 		OwnedBy: "anthropic",
// 	},
// 	"gemini-pro": {
// 		ID:      "gemini-pro",
// 		Object:  "model",
// 		Created: 1704643200,
// 		OwnedBy: "google",
// 	},
// 	"claude-3-opus@20240229": {
// 		ID:      "claude-3-opus@20240229",
// 		Object:  "model",
// 		Created: 1709136000,
// 		OwnedBy: "anthropic",
// 	},
// 	"gpt-4-turbo": {
// 		ID:      "gpt-4-turbo",
// 		Object:  "model",
// 		Created: 1707408000,
// 		OwnedBy: "openai",
// 	},
// 	"gemini-1.5-flash": {
// 		ID:      "gemini-1.5-flash",
// 		Object:  "model",
// 		Created: 1716825600,
// 		OwnedBy: "google",
// 	},
// 	"claude-3-5-sonnet@20240620": {
// 		ID:      "claude-3-5-sonnet@20240620",
// 		Object:  "model",
// 		Created: 1718812800,
// 		OwnedBy: "anthropic",
// 	},
// 	"claude-3-haiku@20240307": {
// 		ID:      "claude-3-haiku@20240307",
// 		Object:  "model",
// 		Created: 1711468800,
// 		OwnedBy: "anthropic",
// 	},
// 	"gpt-3.5-turbo": {
// 		ID:      "gpt-3.5-turbo",
// 		Object:  "model",
// 		Created: 1694793600,
// 		OwnedBy: "openai",
// 	},
// 	"gpt-4": {
// 		ID:      "gpt-4",
// 		Object:  "model",
// 		Created: 1694793600,
// 		OwnedBy: "openai",
// 	},
// }

func InitModels() error {
	var err error
	initModelsOnce.Do(func() {
		SupportedModels = make(map[string]Model)

		// 读取模型配置文件
		data, readErr := assets.Assets.ReadFile("cloud_model.json")
		if readErr != nil {
			err = fmt.Errorf("读取模型配置失败: %v", readErr)
			return
		}

		var config struct {
			Iterable []struct {
				Created struct {
					Value string `json:"value"`
				} `json:"created"`
				Name      string `json:"name"`
				Unique    string `json:"unique"`
				Provider  string `json:"provider"`
				MaxTokens struct {
					Total  int `json:"total"`
					Input  int `json:"input"`
					Output int `json:"output"`
				} `json:"maxTokens"`
			} `json:"iterable"`
		}

		if jsonErr := json.Unmarshal(data, &config); jsonErr != nil {
			err = fmt.Errorf("解析模型配置失败: %v", jsonErr)
			return
		}

		for _, item := range config.Iterable {
			// 解析created时间
			createdTime, _ := time.Parse(time.RFC3339, item.Created.Value)

			// 构建details
			details := map[string]interface{}{
				"name":       item.Name,
				"max_tokens": item.MaxTokens,
			}

			model := Model{
				ID:      item.Unique,
				Object:  "model",
				Created: createdTime.Unix(),
				OwnedBy: strings.ToLower(item.Provider),
				Details: details,
			}

			SupportedModels[item.Unique] = model
		}
	})
	return err
}

func IsModelSupported(modelName string) bool {
	if strings.HasPrefix(modelName, "claude-") && strings.Contains(modelName, "-") && !strings.Contains(modelName, "@") {
		parts := strings.Split(modelName, "-")
		lastPart := parts[len(parts)-1]
		if len(lastPart) == 8 && IsNumeric(lastPart) {
			modelName = strings.Join(parts[:len(parts)-1], "-") + "@" + lastPart
		}
	}
	_, exists := SupportedModels[modelName]
	return exists
}

func IsNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func IsGPTModel(model string) bool {
	// 判断是否为 GPT 模型
	return strings.HasPrefix(model, "gpt-")
}

// NormalizeModelName 标准化模型名称
func NormalizeModelName(m string) string {
	// 如果是 Claude 模型且包含 "-" 而不是 "@"，转换为带 "@" 的格式
	if strings.HasPrefix(m, "claude-") && !strings.Contains(m, "@") {
		parts := strings.Split(m, "-")
		lastPart := parts[len(parts)-1]
		// 检查最后一部分是否是日期格式（8位数字）
		if len(lastPart) == 8 && IsNumeric(lastPart) {
			// 移除最后一部分并用 "@" 重新连接
			newModel := strings.Join(parts[:len(parts)-1], "-") + "@" + lastPart
			return newModel
		}
	}
	return m
}
