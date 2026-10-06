package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/denis-sofonov/go-task-api/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Println("listening on", cfg.HTTPAddr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
