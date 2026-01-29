// Package main запускает sprut daemon.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/udisondev/sprut/internal/appdir"
	"github.com/udisondev/sprut/pkg/config"
	"github.com/udisondev/sprut/pkg/router"
	"gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	configPath := flag.String("config", "", "path to config file (default: XDG config dir)")
	initOnly := flag.Bool("init", false, "initialize app directory and exit")
	flag.Parse()

	// Инициализация директории приложения
	if err := appdir.Init(); err != nil {
		slog.Error("init app directory", "error", err)
		os.Exit(1)
	}

	if *initOnly {
		fmt.Printf("Initialized: %s\n", appdir.Dir())
		fmt.Printf("Config: %s\n", appdir.ConfigPath())
		fmt.Printf("Certs: %s\n", appdir.CertsDir())
		fmt.Printf("Logs: %s\n", appdir.LogsDir())
		return
	}

	if err := run(*configPath); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run(configPath string) error {
	// Загружаем конфигурацию
	var cfg *config.Config
	var err error

	if configPath != "" {
		cfg, err = config.Load(configPath)
	} else {
		cfg, err = config.LoadFromAppDir()
	}
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Настраиваем логирование с ротацией
	setupLogging(cfg.Log)

	slog.Info("sprut starting",
		"config_dir", appdir.Dir(),
		"server_id", cfg.Server.ServerID,
		"address", cfg.Server.Addr(),
	)

	// pprof сервер для профилирования (опционально)
	if pprofAddr := os.Getenv("SPRUT_PPROF"); pprofAddr != "" {
		go func() {
			slog.Info("pprof server started", "addr", pprofAddr)
			if err := http.ListenAndServe(pprofAddr, nil); err != nil {
				slog.Error("pprof server error", "error", err)
			}
		}()
	}

	// Создаём контекст с отменой по сигналам
	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Запускаем роутер
	return router.Run(ctx, cfg)
}

func setupLogging(cfg config.LogConfig) {
	var output io.Writer = os.Stdout

	// Настраиваем ротацию логов если указан файл
	if cfg.File != "" {
		output = &lumberjack.Logger{
			Filename:   cfg.File,
			MaxSize:    100,  // MB
			MaxAge:     7,    // days
			MaxBackups: 5,
			Compress:   true,
			LocalTime:  true,
		}
	}

	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(output, opts)
	} else {
		handler = slog.NewTextHandler(output, opts)
	}

	slog.SetDefault(slog.New(handler))
}
