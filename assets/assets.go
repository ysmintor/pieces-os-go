package assets

import "embed"

//go:embed *.tiktoken *.json
var Assets embed.FS
