package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/cmd"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/config"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/logger"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/platform"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/style"
)

// Build version information injected by goreleaser at build time
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	os.Exit(run())
}

func run() int {
	// Set version info in cmd package
	cmd.Version = version
	cmd.Commit = commit
	cmd.Date = date

	var cli cmd.CLI
	parser, err := cmd.NewParser(&cli)
	if err != nil {
		fmt.Fprintln(os.Stderr, style.FinalError(err.Error()))
		return 1
	}

	kongCtx, err := parser.Parse(os.Args[1:])
	if err != nil {
		parser.FatalIfErrorf(err)
	}

	// Determine debug mode from CLI flag or config
	debugEnabled := cli.Debug || cli.DebugFile != ""
	if !debugEnabled {
		plat := platform.New()
		cfgMgr := config.NewManager(config.ResolveConfigDir(plat))
		if cfg, err := cfgMgr.Load(context.Background()); err == nil && cfg != nil {
			debugEnabled = cfg.Debug
		}
	}

	// Initialize logger
	var slogLogger *slog.Logger
	var logFile *os.File

	if debugEnabled {
		plat := platform.New()
		var logPath string

		if cli.DebugFile != "" {
			logPath = cli.DebugFile
			if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to create log directory: %v\n", err)
				slogLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))
			}
		} else {
			logsDir := plat.GetLogsDir()
			if err := os.MkdirAll(logsDir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to create logs directory: %v\n", err)
				slogLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))
			}
			logPath = filepath.Join(logsDir, fmt.Sprintf("debug-%d.log", time.Now().UnixNano()))
		}

		if slogLogger == nil {
			logFile, err = os.Create(logPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to create log file: %v\n", err)
				slogLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))
			} else {
				fmt.Printf("Debug log: %s\n", logPath)

				replaceAttr := logger.NewRedactionReplaceAttr()
				handler := slog.NewJSONHandler(logFile, &slog.HandlerOptions{
					Level:       slog.LevelDebug,
					ReplaceAttr: replaceAttr,
				})
				slogLogger = slog.New(handler)

				defer func() {
					_ = logFile.Close() //nolint:errcheck // cleanup
				}()

				slogLogger.Info("claudectl started",
					"version", version,
					"commit", commit,
					"args", os.Args[1:],
				)
			}
		}
	} else {
		slogLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}

	// Create execution context
	baseCtx := logger.WithLogger(context.Background(), slogLogger)
	execCtx := &cmd.Context{
		CLI:     &cli,
		Context: baseCtx,
		Logger:  slogLogger,
	}

	// Run command
	startTime := time.Now()
	slogLogger.Info("command started",
		"command", kongCtx.Command(),
		"args", os.Args[1:],
	)

	err = kongCtx.Run(execCtx)
	duration := time.Since(startTime)

	slogLogger.Info("command completed",
		"command", kongCtx.Command(),
		"duration_ms", duration.Milliseconds(),
	)

	if err != nil {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, style.FinalError(err.Error()))
		return 1
	}
	return 0
}
