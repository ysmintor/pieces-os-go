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
