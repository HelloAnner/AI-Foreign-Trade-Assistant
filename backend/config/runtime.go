package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	AppDirName       = ".foreign_trade"
	legacyAppDirName = "foreign_trade"
	ConfigFileName   = "config.json"
	DatabaseFile     = "app.db"
	LogDirName       = "logs"
	CacheDirName     = "cache"
	ExportsDirName   = "exports"
	DefaultHTTPAddr  = "0.0.0.0:25000"
	httpAddrEnvKey   = "APP_HTTP_ADDR"
	httpPortEnvKey   = "APP_PORT"
)

// HTTPAddr 返回 HTTP 服务应绑定的地址，优先读取环境变量。
func HTTPAddr() string {
	if addr := strings.TrimSpace(os.Getenv(httpAddrEnvKey)); addr != "" {
		return addr
	}

	if port := strings.TrimSpace(os.Getenv(httpPortEnvKey)); port != "" {
		trimmed := strings.TrimPrefix(port, ":")
		if _, err := strconv.Atoi(trimmed); err == nil {
			return fmt.Sprintf("0.0.0.0:%s", trimmed)
		}
	}

	return DefaultHTTPAddr
}

// Paths keeps resolved filesystem locations the app relies on.
type Paths struct {
	RootDir    string
	ConfigFile string
	DBFile     string
	LogDir     string
	CacheDir   string
	ExportsDir string
}

// ResolvePaths builds the set of directories under the configured base.
func ResolvePaths(homeDir string) (*Paths, error) {
	if homeDir == "" {
		return nil, fmt.Errorf("empty home directory")
	}
	root := filepath.Join(homeDir, AppDirName)
	return &Paths{
		RootDir:    root,
		ConfigFile: filepath.Join(root, ConfigFileName),
		DBFile:     filepath.Join(root, DatabaseFile),
		LogDir:     filepath.Join(root, LogDirName),
		CacheDir:   filepath.Join(root, CacheDirName),
		ExportsDir: filepath.Join(root, ExportsDirName),
	}, nil
}

// Ensure lays out the directory skeleton if missing and creates the config file when needed.
func Ensure(paths *Paths) error {
	if paths == nil {
		return fmt.Errorf("paths is nil")
	}

	// Attempt to migrate legacy directory if present.
	if _, err := os.Stat(paths.RootDir); os.IsNotExist(err) {
		parent := filepath.Dir(paths.RootDir)
		legacyRoot := filepath.Join(parent, legacyAppDirName)
		if info, legacyErr := os.Stat(legacyRoot); legacyErr == nil && info.IsDir() {
			if renameErr := os.Rename(legacyRoot, paths.RootDir); renameErr != nil {
				// fallback to creating fresh structure below
			}
		}
	}

	dirs := []string{
		paths.RootDir,
		paths.LogDir,
		paths.CacheDir,
		paths.ExportsDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
	}

	if _, err := os.Stat(paths.ConfigFile); os.IsNotExist(err) {
		if err := os.WriteFile(paths.ConfigFile, []byte("{}"), 0o644); err != nil {
			return fmt.Errorf("create config file: %w", err)
		}
	}

	return nil
}
