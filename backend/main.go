package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/anner/ai-foreign-trade-assistant/backend/api"
	appconfig "github.com/anner/ai-foreign-trade-assistant/backend/config"
	"github.com/anner/ai-foreign-trade-assistant/backend/logging"
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

	// Initialize logging to daily-rotated files under ~/.foreign_trade/logs
	if err := logging.Setup(paths); err != nil {
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

	automationRunner := task.NewAutomationRunner(bundle.Automation)
	if automationRunner != nil {
		automationRunner.Start(ctx)
		defer automationRunner.Stop()
	}

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

	displayAddr, err := resolveDisplayAddr(server.Addr)
	if err != nil {
		log.Printf("resolve display address: %v", err)
		displayAddr = server.Addr
	}
	displayURL := fmt.Sprintf("http://%s", displayAddr)
	log.Printf("AI 外贸客户开发助手服务已启动，访问 %s", displayURL)

	if shouldAutoOpenBrowser() {
		go func() {
			time.Sleep(500 * time.Millisecond)
			openBrowser(displayURL)
		}()
	}

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

func resolveDisplayAddr(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}

	if lanIP, err := firstLANIPv4(); err == nil && lanIP != "" {
		host = lanIP
	} else if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}

	return net.JoinHostPort(host, port), nil
}

func firstLANIPv4() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range ifaces {
		if (iface.Flags&net.FlagUp) == 0 || (iface.Flags&net.FlagLoopback) != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil {
				continue
			}
			ip = ip.To4()
			if ip == nil || ip.IsLoopback() {
				continue
			}
			return ip.String(), nil
		}
	}

	return "", fmt.Errorf("no LAN IPv4 address found")
}

func shouldAutoOpenBrowser() bool {
	val := strings.TrimSpace(os.Getenv("FOREIGN_TRADE_NO_BROWSER"))
	if val == "" {
		return true
	}
	val = strings.ToLower(val)
	return val != "1" && val != "true" && val != "yes"
}

func openBrowser(url string) {
	if url == "" {
		return
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	if err := cmd.Start(); err != nil {
		log.Printf("自动打开浏览器失败: %v", err)
	}
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
