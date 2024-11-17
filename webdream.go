// Copyright 2024 Fabian Bergström
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"log/slog"
	"net/http"
	"os"
	"text/template"

	"github.com/fabjan/webdream/dream"
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

	result, cacheHit := dream.GetCachedResponse(reqPath)
	var err error

	if !cacheHit {
		result, err = groq.Dream(reqPath)
		if result != nil {
			dream.CacheResponse(reqPath, result)
		}
	}

	if result == nil {
		if err != nil {
			slog.Error("Cannot dream", "err", err)
			http.NotFound(w, r)
			return
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(result.Status)
	_, err = w.Write([]byte(result.Body))
	if err != nil {
		slog.Error("Failed to write response", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Request handled", "path", reqPath, "status", result.Status, "cacheHit", cacheHit)
}
