package dream

import (
	_ "embed"
)

//go:embed system-prompt.txt
var SystemPrompt string

type Response struct {
	Headers map[string]string `json:"headers"`
	Status  int               `json:"status"`
	Body    string            `json:"body"`
}
