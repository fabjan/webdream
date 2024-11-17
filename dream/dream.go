package dream

import (
	_ "embed"
)

//go:embed system-prompt.txt
var SystemPrompt string

type Response struct {
	Status int    `json:"status"`
	Body   string `json:"body"`
}
}
