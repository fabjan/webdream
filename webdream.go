package main

import (
	"log/slog"
	"net/http"
	"os"
	"text/template"

	"github.com/fabjan/webdream/groq"
	"github.com/fabjan/webdream/metrics"
)

func checkRateLimit() bool {
	// half-arbitrary combination of Groq limits
	if 30 <= metrics.CountRequestsInLastMinute() {
		return true
	}
	if 14400 <= metrics.CountRequestsInLastDay() {
		return true
	}
	if 5000 <= metrics.CountTokensInLastMinute() {
		return true
	}
	if 500000 <= metrics.CountTokensInLastDay() {
		return true
	}

	return false
}

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":3000"
		port := os.Getenv("PORT")
		if port != "" {
			addr = ":" + port
		}
	}

	http.Handle("/metrics", metrics.Handler())
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
	stats := map[string]int{
		"RequestsLastMinute": metrics.CountRequestsInLastMinute(),
		"RequestsLastDay":    metrics.CountRequestsInLastDay(),
		"TokensLastMinute":   metrics.CountTokensInLastMinute(),
		"TokensLastDay":      metrics.CountTokensInLastDay(),
	}
	err = t.Execute(w, stats)
	if err != nil {
		slog.Error("Failed to execute template", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func dreamHandler(w http.ResponseWriter, r *http.Request) {

	if checkRateLimit() {
		http.Error(w, "Rate limited", http.StatusTooManyRequests)
		return
	}

	reqPath := r.URL.Path
	result, err := groq.Dream(reqPath)
	if err != nil {
		slog.Error("Cannot dream", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(result.Status)
	_, err = w.Write([]byte(result.Body))
	if err != nil {
		slog.Error("Failed to write response", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Request handled", "path", reqPath, "status", result.Status)
}
