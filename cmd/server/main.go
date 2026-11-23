package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"yoomates/internal/config"
)

type HealthResponse struct {
	Status    string `json:"status"`
	Engine    string `json:"engine"`
	Env       string `json:"env"`
	Timestamp string `json:"timestamp"`
}

func HealthHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(HealthResponse{
			Status:    "ok",
			Engine:    "yoomates-go",
			Env:       cfg.Env,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
	}
}

func SetupRoutes(cfg *config.Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", HealthHandler(cfg))
	return mux
}

func main() {
	cfg := config.Load()

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      SetupRoutes(cfg),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("[yoomates-server] Starting (%s) on http://localhost:%s\n", cfg.Env, cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[yoomates-server] Listen error: %v\n", err)
		}
	}()

	<-stop
	log.Println("[yoomates-server] Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("[yoomates-server] Server forced shutdown: %v\n", err)
	}
	fmt.Println("[yoomates-server] Stopped cleanly.")
}
