package main

import (
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"
)

//go:embed ui
var uiFiles embed.FS

func cmdServe(db *sql.DB, port int) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /items", func(w http.ResponseWriter, r *http.Request) {
		items, err := list(db)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		json.NewEncoder(w).Encode(items)
	})

	mux.HandleFunc("POST /items", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Source string `json:"source"`
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		item, err := getItem(body.Source)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		if err := add(db, item); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		w.WriteHeader(http.StatusCreated)
	})

	mux.HandleFunc("DELETE /items/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))

		if err := del(db, id); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /items/{id}/html", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))

		page, err := getHTML(db, id)
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// sandbox saved pages
		w.Header().Set("Content-Security-Policy", "sandbox")
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(page))
	})

	mux.HandleFunc("GET /search", func(w http.ResponseWriter, r *http.Request) {
		res, err := search(db, r.URL.Query().Get("q"))
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		json.NewEncoder(w).Encode(res)
	})

	// Serve the frontend though go and static html + css + js files
	ui, err := fs.Sub(uiFiles, "ui")
	if err != nil {
		log.Fatal(err)
	}

	mux.Handle("GET /", http.FileServerFS(ui))

	addr := fmt.Sprintf("localhost:%d", port)
	log.Fatal(http.ListenAndServe(addr, localOnly(mux)))
}

// accept only localhost requests
func localOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Host, "localhost:") && !strings.HasPrefix(r.Host, "127.0.0.1:") {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		if r.Method == http.MethodPost && !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			http.Error(w, "json only", http.StatusUnsupportedMediaType)
			return
		}

		next.ServeHTTP(w, r)
	})
}
