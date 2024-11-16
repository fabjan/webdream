package main

import (
	"fmt"
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

	err := http.ListenAndServe(addr, nil)
	if err != nil {
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.Execute(w, nil)
}

func dreamHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %s!", r.URL.Path[1:])
}
