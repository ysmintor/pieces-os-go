package tokenizer

import "strings"

type modelEncoding struct {
	prefix   string
	encoding string
}

var modelPrefixToEncoding = []modelEncoding{
	{"o1-", "o200k_base"},
	{"chatgpt-4o-", "o200k_base"},
	{"gpt-4o-", "o200k_base"},
	{"gpt-4-", "cl100k_base"},
	{"gpt-3.5-turbo-", "cl100k_base"},
	{"gpt-35-turbo-", "cl100k_base"},
}

var modelToEncoding = []modelEncoding{
	{"gpt-4o", "o200k_base"},
	{"gpt-4", "cl100k_base"},
	{"gpt-3.5-turbo", "cl100k_base"},
	{"gpt-3.5", "cl100k_base"},
	{"gpt-35-turbo", "cl100k_base"},
}

// getEncoding 返回模型对应的编码，如果未匹配则返回空字符串
func getEncoding(model string) string {
	// 直接匹配完整模型名
	for _, m := range modelToEncoding {
		if m.prefix == model {
			return m.encoding
		}
	}
	// 匹配模型前缀
	for _, m := range modelPrefixToEncoding {
		if strings.HasPrefix(model, m.prefix) {
			return m.encoding
		}
	}
	// 未匹配时返回空字符串
	return ""
}
