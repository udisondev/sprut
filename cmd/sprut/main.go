// Package main запускает goro daemon.
package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/udisondev/sprut/internal/config"
	"github.com/udisondev/sprut/internal/router"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	if err := run(*configPath); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run(configPath string) error {
	// Загружаем конфигурацию
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	// Настраиваем логирование
	setupLogging(cfg.Log)

	// pprof сервер для профилирования (опционально)
	// Использование: GORO_PPROF=localhost:6060 ./gorod
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

	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: level}

	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}
