package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"text/template"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":3000"
		port := os.Getenv("PORT")
		if port != "" {
			addr = ":" + port
		}
	}

	http.HandleFunc("/", rootHandler)

	slog.Info("Starting server", "addr", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		slog.Error("Failed to start server", "err", err)
		panic(err)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		dreamHandler(w, r)
		return
	}

	t, err := template.ParseFiles("public/index.html")
	if err != nil {
		slog.Error("Failed to parse template", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.Execute(w, nil)
}

type GroqChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GroqRequest struct {
	Model    string            `json:"model"`
	Messages []GroqChatMessage `json:"messages"`
}

type GroqResponse struct {
	Choices []struct {
		Message GroqChatMessage `json:"message"`
	} `json:"choices"`
}

type WebDreamResponse struct {
	Headers map[string]string `json:"headers"`
	Status  int               `json:"status"`
	Body    string            `json:"body"`
}

func dreamHandler(w http.ResponseWriter, r *http.Request) {
	reqPath := r.URL.Path
	apiKey := os.Getenv("GROQ_API_KEY")

	reqData := GroqRequest{
		Model: "llama3-8b-8192",
		Messages: []GroqChatMessage{
			{
				Role:    "system",
				Content: "You are the backend of a web service. You will receive each request as a JSON object with a 'headers', 'path', and 'body' field. You should respond with a JSON object the service can parse to respond. Your JSON has the properties 'headers', 'status', and 'body'. Responses have a handful of paragraphs. You can render links, but only to the same origin.",
			},
			{
				Role: "user",
				Content: `{
					"headers": {
						"Content-Type": "text/html"
					},
					"path": "` + reqPath + `",
				}`,
			},
		},
	}

	bodyBytes := new(bytes.Buffer)
	json.NewEncoder(bodyBytes).Encode(reqData)

	chatCompletionsURL := "https://api.groq.com/openai/v1/chat/completions"
	req, err := http.NewRequest("POST", chatCompletionsURL, bodyBytes)
	if err != nil {
		slog.Error("Failed to create request", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Failed to make request", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// read the response body
	var respData GroqResponse
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		slog.Error("Failed to decode response", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// parse the wrapped JSON inside
	firstChoice := respData.Choices[0].Message.Content
	var webDreamResp WebDreamResponse
	err = json.Unmarshal([]byte(firstChoice), &webDreamResp)
	if err != nil {
		slog.Error("Failed to decode inner response", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for k, v := range webDreamResp.Headers {
		w.Header().Set(k, v)
	}
	w.WriteHeader(webDreamResp.Status)
	_, err = w.Write([]byte(webDreamResp.Body))
	if err != nil {
		slog.Error("Failed to write response", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Responded to request", "path", reqPath)
}
