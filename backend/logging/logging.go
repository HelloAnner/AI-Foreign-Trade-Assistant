package logging

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	appconfig "github.com/anner/ai-foreign-trade-assistant/backend/config"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
)

var logger *slog.Logger

// Setup configures application logging.
// - Writes logs to ~/.foreign_trade/logs
// - Rotates daily (by date) and compresses previous files
// - Defaults to info level for slog
func Setup(paths *appconfig.Paths) error {
	if paths == nil {
		return fmt.Errorf("logging: nil paths")
	}

	if err := os.MkdirAll(paths.LogDir, 0o755); err != nil {
		return fmt.Errorf("logging: ensure log dir: %w", err)
	}

	// Current log symlink name and rotation pattern
	linkName := filepath.Join(paths.LogDir, "app.log")
	pattern := filepath.Join(paths.LogDir, "app.%Y-%m-%d.log")

	rl, err := rotatelogs.New(
		pattern,
		rotatelogs.WithLinkName(linkName),
		rotatelogs.WithRotationTime(24*time.Hour),
		rotatelogs.WithHandler(rotatelogs.HandlerFunc(func(e rotatelogs.Event) {
			// When rotation happens, compress the previous file asynchronously.
			if e.Type() == rotatelogs.FileRotatedEventType {
				prev := e.(*rotatelogs.FileRotatedEvent).PreviousFile()
				if prev == "" || strings.HasSuffix(prev, ".gz") {
					return
				}
				go func(src string) {
					// Best-effort compression; on error, write to stderr.
					if err := gzipFile(src, src+".gz"); err != nil {
						_, _ = fmt.Fprintf(os.Stderr, "compress log error: %v\n", err)
						return
					}
					_ = os.Remove(src)
				}(prev)
			}
		})),
	)
	if err != nil {
		return fmt.Errorf("logging: init rotator: %w", err)
	}

	// Route the standard library logger to both rotating file and stdout.
	mw := io.MultiWriter(rl, os.Stdout)
	log.SetOutput(mw)
	log.SetFlags(log.LstdFlags)

	// Configure slog with default info level, also writing to rotating writer.
	level := slog.LevelInfo
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL"))) {
	case "debug":
		level = slog.LevelDebug
	case "", "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	case "off", "none":
		// Drop everything by setting a very high threshold.
		level = slog.Level(100)
	}

	handler := slog.NewTextHandler(mw, &slog.HandlerOptions{Level: level})
	logger = slog.New(handler)

	return nil
}

// Logger returns the structured logger (slog) instance.
func Logger() *slog.Logger {
	if logger == nil {
		// Fallback to a no-op logger if Setup wasn't called.
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return logger
}

func gzipFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	// Ensure cleanup on error
	defer func() {
		_ = out.Close()
		if err != nil {
			_ = os.Remove(dst)
		}
	}()

	gz := gzip.NewWriter(out)
	gz.Name = filepath.Base(src)
	if _, err = io.Copy(gz, in); err != nil {
		_ = gz.Close()
		return err
	}
	if err = gz.Close(); err != nil {
		return err
	}
	return out.Close()
}
