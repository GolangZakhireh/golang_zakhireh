package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golangzakhireh/internal/config"
	"golangzakhireh/internal/dashboard"
	"golangzakhireh/internal/proxy"
	"golangzakhireh/internal/server"
	"golangzakhireh/internal/storage"
	"golangzakhireh/internal/upload"
)

func main() {
	// Ensure /logs directory exists
	logDir := "logs"
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		if err := os.Mkdir(logDir, 0755); err != nil {
			log.Fatalf("Failed to create log directory: %v", err)
		}
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logFileName := filepath.Join(logDir, fmt.Sprintf("golangzakhireh_%s.log", timestamp))
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	// Set up multi-writer for logging to both terminal and log file
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)

	// Print ASCII banner
	log.Println(`
	 ██████╗  ██████╗ ██╗      █████╗ ███╗   ██╗ ██████╗     ███████╗ █████╗ ██╗  ██╗██╗  ██╗██╗██████╗ ███████╗██╗  ██╗
	██╔════╝ ██╔═══██╗██║     ██╔══██╗████╗  ██║██╔════╝     ╚══███╔╝██╔══██╗██║ ██╔╝██║  ██║██║██╔══██╗██╔════╝██║  ██║
	██║  ███╗██║   ██║██║     ███████║██╔██╗ ██║██║  ███╗      ███╔╝ ███████║█████╔╝ ███████║██║██████╔╝█████╗  ███████║
	██║   ██║██║   ██║██║     ██╔══██║██║╚██╗██║██║   ██║     ███╔╝  ██╔══██║██╔═██╗ ██╔══██║██║██╔══██╗██╔══╝  ██╔══██║
	╚██████╔╝╚██████╔╝███████╗██║  ██║██║ ╚████║╚██████╔╝    ███████╗██║  ██║██║  ██╗██║  ██║██║██║  ██║███████╗██║  ██║
	 ╚═════╝  ╚═════╝ ╚══════╝╚═╝  ╚═╝╚═╝  ╚═══╝ ╚═════╝     ╚══════╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝                                                                                                             
	`)

	// Load configs from Environment Variables
	cfg := config.Load()

	// Initialize storage
	store := storage.NewFSBackend(cfg.DataDir)
	if err := store.Init(); err != nil {
		slog.Error("failed to init storage", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()

	// Validator
	validator := proxy.NewValidator(cfg.NoSumDB, cfg.AllowList, cfg.DenyList)

	// Handlers
	proxyHandler := proxy.NewProxyHandler(store, cfg.UpstreamProxy, validator)
	uploadHandler := upload.NewUploadHandler(store)
	dashboardHandler := dashboard.NewDashboardHandler(store)

	// Routes
	// Routes
	mux.Handle("/upload", uploadHandler)

	// Intelligent routing: If path contains module version markers, send to proxy.
	// Otherwise, serve the dashboard.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Module requests contain /@v/ or /@latest
		if proxy.IsModuleRequest(r.URL.Path) {
			// Strip /proxy/ prefix if present (for explicit proxy requests)
			if PathHasProxyPrefix(r.URL.Path) {
				http.StripPrefix("/proxy", proxyHandler).ServeHTTP(w, r)
				return
			}
			// Direct usage as GOPROXY
			proxyHandler.ServeHTTP(w, r)
			return
		}

		// Specific proxy prefix check (for when users explicitly use /proxy/ base)
		if PathHasProxyPrefix(r.URL.Path) {
			http.StripPrefix("/proxy", proxyHandler).ServeHTTP(w, r)
			return
		}

		// Default to dashboard
		dashboardHandler.ServeHTTP(w, r)
	})

	log.Printf("🚀 GolangZakhireh is running on port: %s", cfg.Port)
	log.Printf("📊 Dashboard: http://localhost%s", cfg.Port)
	log.Printf("📦 Data directory: %s", cfg.DataDir)
	log.Fatal(http.ListenAndServe(cfg.Port, server.AccessLogger(mux)))
}

func PathHasProxyPrefix(path string) bool {
	return len(path) >= 7 && path[:7] == "/proxy/"
}
