package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"yoomates/internal/auth"
	"yoomates/internal/config"
	"yoomates/internal/db"
	"yoomates/internal/feed"
	"yoomates/internal/hubs"
	"yoomates/internal/space"
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

func SetupRoutes(cfg *config.Config, authMgr *auth.Manager, hub *hubs.Hub, viewEngine *hubs.ViewEngine, uploader *hubs.Uploader, optionalEngines ...any) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", HealthHandler(cfg))

	// Resolve static directory (supports both project root and test execution)
	staticDir := "web/static"
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		staticDir = "../../web/static"
	}

	// Static assets file server
	staticFS := http.FileServer(http.Dir(staticDir))
	mux.Handle("/static/", http.StripPrefix("/static/", staticFS))

	// Service Worker for 0ms offline resilience (scope: /)
	mux.HandleFunc("/sw.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Header().Set("Service-Worker-Allowed", "/")
		swPath := filepath.Join(staticDir, "sw.js")
		http.ServeFile(w, r, swPath)
	})

	// Attachment file uploads server & API endpoint
	if uploader != nil {
		mux.HandleFunc("/api/hubs/upload", hubs.UploadHandler(uploader, authMgr))
		mux.Handle("/uploads/", http.StripPrefix("/uploads", hubs.ServeFilesHandler(uploader)))
	}

	// Root redirect: Hubs is the default primary view
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

	var feedEngine *feed.ViewEngine
	var spaceEngine *space.ViewEngine
	for _, e := range optionalEngines {
		switch v := e.(type) {
		case *feed.ViewEngine:
			feedEngine = v
		case *space.ViewEngine:
			spaceEngine = v
		}
	}

	if feedEngine != nil {
		var globalHandler http.Handler = http.HandlerFunc(feedEngine.GlobalHandler)
		if authMgr != nil {
			globalHandler = authMgr.SessionMiddleware(globalHandler)
		}
		mux.Handle("/global", globalHandler)

		// Post creation & stream pagination
		mux.HandleFunc("/global/posts", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				feedEngine.CreatePostHandler(w, r)
			} else {
				feedEngine.PostsStreamHandler(w, r)
			}
		})

		// Dynamic sub-routes: /global/posts/{id}/like and /global/posts/{id}/comments
		mux.HandleFunc("/global/posts/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/like") {
				feedEngine.LikeHandler(w, r)
				return
			}
			if strings.Contains(r.URL.Path, "/comments") {
				feedEngine.CommentsHandler(w, r)
				return
			}
			http.NotFound(w, r)
		})

		// Poll vote handler: /global/polls/{id}/vote
		mux.HandleFunc("/global/polls/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/vote") {
				feedEngine.PollVoteHandler(w, r)
				return
			}
			http.NotFound(w, r)
		})
	}

	if spaceEngine != nil {
		var mySpaceHandler http.Handler = http.HandlerFunc(spaceEngine.MySpaceHandler)
		if authMgr != nil {
			mySpaceHandler = authMgr.SessionMiddleware(mySpaceHandler)
		}
		mux.Handle("/myspace", mySpaceHandler)

		// Document creation & autosave
		mux.HandleFunc("/myspace/documents", spaceEngine.CreateDocumentHandler)
		mux.HandleFunc("/myspace/documents/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/autosave") {
				spaceEngine.AutosaveHandler(w, r)
				return
			}
			if strings.HasSuffix(r.URL.Path, "/status") {
				spaceEngine.StatusHandler(w, r)
				return
			}
			spaceEngine.GetDocumentHandler(w, r)
		})

		// Workspace views: Kanban board & List table
		mux.HandleFunc("/myspace/view/kanban", spaceEngine.KanbanViewHandler)
		mux.HandleFunc("/myspace/view/list", spaceEngine.ListViewHandler)
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

	// Initialize real-time WebSocket Hub with MySQL repository
	repo := hubs.NewRepository(pool)
	hub := hubs.NewHub(repo)
	go hub.Run()
	log.Println("[yoomates-hubs] Real-time WebSocket hub active.")

	// Initialize HTML template view engine
	viewEngine, err := hubs.NewViewEngine("web/templates", pool)
	if err != nil {
		log.Printf("[yoomates-views] Warning: Could not initialize template view engine (%v)\n", err)
	} else {
		log.Println("[yoomates-views] HTML templates loaded successfully.")
	}

	// Initialize local/S3 multipart attachment upload pipeline
	uploader, err := hubs.NewUploader("data/uploads", hubs.DefaultMaxUploadSize, "/uploads")
	if err != nil {
		log.Printf("[yoomates-upload] Warning: Could not initialize attachment pipeline (%v)\n", err)
	} else {
		log.Println("[yoomates-upload] Multipart attachment upload pipeline active.")
	}

	// Initialize Global Feed repository & view engine
	feedRepo := feed.NewRepository(pool)
	feedEngine, err := feed.NewViewEngine("web/templates", feedRepo)
	if err != nil {
		log.Printf("[yoomates-feed] Warning: Could not initialize feed view engine (%v)\n", err)
	} else {
		log.Println("[yoomates-feed] Global Feed engine active.")
	}

	// Initialize My Space workspace engine
	spaceRepo := space.NewRepository(pool)
	spaceEngine, err := space.NewViewEngine("web/templates", spaceRepo)
	if err != nil {
		log.Printf("[yoomates-space] Warning: Could not initialize space view engine (%v)\n", err)
	} else {
		log.Println("[yoomates-space] My Space workspace engine active.")
	}

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      SetupRoutes(cfg, authMgr, hub, viewEngine, uploader, feedEngine, spaceEngine),
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
