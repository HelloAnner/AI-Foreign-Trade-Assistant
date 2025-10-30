package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/anner/ai-foreign-trade-assistant/backend/api"
	appconfig "github.com/anner/ai-foreign-trade-assistant/backend/config"
	"github.com/anner/ai-foreign-trade-assistant/backend/services"
	"github.com/anner/ai-foreign-trade-assistant/backend/store"
	"github.com/anner/ai-foreign-trade-assistant/backend/task"
)

//go:embed all:static
var staticFS embed.FS

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		log.Fatalf("startup error: %v", err)
	}
}

func run(ctx context.Context) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	paths, err := appconfig.ResolvePaths(homeDir)
	if err != nil {
		return err
	}

	if err := appconfig.Ensure(paths); err != nil {
		return err
	}

	dataStore, err := store.Open(paths.DBFile)
	if err != nil {
		return err
	}
	defer dataStore.Close()

	if err := dataStore.InitSchema(ctx); err != nil {
		return err
	}

	bundle := services.NewBundle(services.Options{Store: dataStore})

	runner := task.NewRunner(dataStore, bundle.Scheduler)
	runner.Start(ctx)

	handlers := &api.Handlers{
		Store:         dataStore,
		ServiceBundle: bundle,
	}

	staticContent, err := fs.Sub(staticFS, "static")
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", api.Router(handlers))
	mux.Handle("/", spaHandler(staticContent))

	server := &http.Server{
		Addr:              appconfig.DefaultHTTPAddr,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("server shutdown error: %v", err)
		}
	}()

	log.Printf("AI 外贸客户开发助手服务已启动，监听 http://%s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func spaHandler(content fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(content))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" || path == "index.html" {
			fileServer.ServeHTTP(w, r)
			return
		}
		if _, err := content.Open(path); err != nil {
			r2 := *r
			r2.URL.Path = "/index.html"
			fileServer.ServeHTTP(w, &r2)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}
