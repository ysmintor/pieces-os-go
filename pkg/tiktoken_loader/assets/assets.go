package assets

import "embed"

//go:embed *.tiktoken tokenizer.json
var Assets embed.FS
