package server

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/alexandrmotologa/queryproof/internal/config"
	"github.com/alexandrmotologa/queryproof/internal/engine"
	"github.com/alexandrmotologa/queryproof/internal/metrics"
)

// Server represents the queryproof HTTP server.
type Server struct {
	config  *config.ServerConfig
	httpSrv *http.Server
	handler *APIHandler
	webFS   fs.FS
}

// NewServer initializes the HTTP server with API routes and static asset serving.
func NewServer(
	cfg *config.ServerConfig,
	eng *engine.VerificationEngine,
	cat *config.Catalog,
	reg *metrics.Registry,
	secretKey string,
	webAssets fs.FS,
) *Server {
	if cfg == nil {
		cfg = &config.ServerConfig{Port: 7373, Host: "127.0.0.1", CORS: true}
	}

	api := NewAPIHandler(eng, cat, reg, secretKey)
	mux := http.NewServeMux()

	// Register API endpoints
	mux.HandleFunc("/api/verify", api.HandleVerify)
	mux.HandleFunc("/api/verify-passport", api.HandleVerifyPassport)
	mux.HandleFunc("/api/catalog", api.HandleCatalog)
	mux.HandleFunc("/api/metrics", api.HandleMetrics)
	mux.HandleFunc("/api/scenarios", api.HandleScenarios)
	mux.HandleFunc("/api/health", api.HandleHealth)

	// Static frontend assets serving
	if webAssets != nil {
		fileServer := http.FileServer(http.FS(webAssets))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}
			// Attempt to serve file, fallback to index.html for SPA routing
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" {
				path = "index.html"
			}
			f, err := webAssets.Open(path)
			if err != nil {
				// Fallback to index.html
				indexData, errIndex := fs.ReadFile(webAssets, "index.html")
				if errIndex == nil {
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write(indexData)
					return
				}
				http.NotFound(w, r)
				return
			}
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
		})
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>queryproof BI Auditor API</title></head>
<body style="font-family: sans-serif; padding: 40px; background: #0f172a; color: #f8fafc;">
  <h1>queryproof API Server Running</h1>
  <p>Status: Healthy. Web assets not embedded in this build mode.</p>
  <p>Available endpoints: <code>/api/verify</code>, <code>/api/catalog</code>, <code>/api/metrics</code>, <code>/api/scenarios</code>, <code>/api/health</code></p>
</body>
</html>`))
				return
			}
			http.NotFound(w, r)
		})
	}

	// CORS wrapper
	var rootHandler http.Handler = mux
	if cfg.CORS {
		rootHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			mux.ServeHTTP(w, r)
		})
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      rootHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	return &Server{
		config:  cfg,
		httpSrv: srv,
		handler: api,
		webFS:   webAssets,
	}
}

// Start begins listening and serving HTTP traffic.
func (s *Server) Start() error {
	log.Printf("[queryproof] BI Auditor Cockpit listening on http://%s\n", s.httpSrv.Addr)
	if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}
