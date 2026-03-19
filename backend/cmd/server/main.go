package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"anonchat/backend/internal/config"
	"anonchat/backend/internal/handler"
	"anonchat/backend/internal/ratelimit"
	"anonchat/backend/internal/redisclient"
	"anonchat/backend/internal/room"

	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}

	rc := redisclient.New(rdb)
	mgr := room.NewManager(rc, rc, 50)
	msgLim := ratelimit.NewMessageLimiter(10, time.Second)
	ipLim := ratelimit.NewIPRoomLimiter(10, time.Minute)

	h := handler.New(mgr, msgLim, ipLim, cfg)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: h,
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
		<-sigCh
		slog.Info("shutting down")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		h.Shutdown(shutdownCtx)
		srv.Shutdown(shutdownCtx)
	}()

	slog.Info("starting server", "port", cfg.Port)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
