package dream

type Response struct {
	Headers map[string]string `json:"headers"`
	Status  int               `json:"status"`
	Body    string            `json:"body"`
}
