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

	"yoomates/internal/auth"
	"yoomates/internal/config"
	"yoomates/internal/db"
	"yoomates/internal/hubs"
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

func SetupRoutes(cfg *config.Config, authMgr *auth.Manager, hub *hubs.Hub, viewEngine *hubs.ViewEngine) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", HealthHandler(cfg))

	// Static assets file server
	staticFS := http.FileServer(http.Dir("web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", staticFS))

	// Root redirect: Hubs (Discord chat) is the default primary view
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/hubs", http.StatusTemporaryRedirect)
	})

	if viewEngine != nil {
		var hubsHandler http.Handler = http.HandlerFunc(viewEngine.HubsHandler)
		if authMgr != nil {
			hubsHandler = authMgr.SessionMiddleware(hubsHandler)
		}
		mux.Handle("/hubs", hubsHandler)
	}

	if authMgr != nil {
		mux.HandleFunc("/api/auth/register", authMgr.RegisterHandler)
		mux.HandleFunc("/api/auth/login", authMgr.LoginHandler)
		mux.HandleFunc("/api/auth/logout", authMgr.LogoutHandler)
		mux.Handle("/api/auth/me", authMgr.SessionMiddleware(auth.RequireAuth(http.HandlerFunc(authMgr.MeHandler))))
	}

	if hub != nil {
		mux.HandleFunc("/ws", hubs.WSHandler(hub, authMgr))
	}

	return mux
}

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize connection pool
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Printf("[yoomates-db] Warning: Database pool uninitialized (%v). Running with mock/offline state.\n", err)
	} else {
		defer pool.Close()
		log.Println("[yoomates-db] MySQL connection pool active.")
	}

	var authMgr *auth.Manager
	if pool != nil {
		authMgr = auth.NewManager(pool, cfg)
	}

	// Initialize real-time WebSocket Hub
	hub := hubs.NewHub()
	go hub.Run()
	log.Println("[yoomates-hubs] Real-time WebSocket hub active.")

	// Initialize HTML template view engine
	viewEngine, err := hubs.NewViewEngine("web/templates", pool)
	if err != nil {
		log.Printf("[yoomates-views] Warning: Could not initialize template view engine (%v)\n", err)
	} else {
		log.Println("[yoomates-views] HTML templates loaded successfully.")
	}

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      SetupRoutes(cfg, authMgr, hub, viewEngine),
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

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("[yoomates-server] Server forced shutdown: %v\n", err)
	}
	fmt.Println("[yoomates-server] Stopped cleanly.")
}
