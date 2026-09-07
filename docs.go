package main

import (
	"embed"
	"net/http"
)

//go:embed docs/index.html
var docsHTML embed.FS

func docsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	data, err := docsHTML.ReadFile("docs/index.html")
	if err != nil {
		http.Error(w, "documentation unavailable", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(data)
}
