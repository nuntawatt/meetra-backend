// Package main is the application entry point.
// It loads env vars, bootstraps the app, starts the HTTP server,
// and handles graceful shutdown on SIGINT / SIGTERM.
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/go-wego/wego/internal/bootstrap"
	"github.com/go-wego/wego/internal/config"
	deliveryHTTP "github.com/go-wego/wego/internal/delivery/http"
	"github.com/go-wego/wego/pkg/logger"
)

func main() {
	// Load .env file if present (ignored in production; env vars come from the host)
	_ = godotenv.Load()

	// Bootstrap: wire all dependencies
	app, err := bootstrap.New()
	if err != nil {
		// Logger might not be ready yet — fall back to stdlib
		panic("bootstrap failed: " + err.Error())
	}
	log := app.Logger
	defer logger.Sync()

	// Build the Gin router
	appCfg := config.LoadApp()
	router := deliveryHTTP.NewRouter(*app.Router)

	// Configure the HTTP server
	srv := &http.Server{
		Addr:         ":" + appCfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start the server in a goroutine so we can listen for shutdown signals
	go func() {
		log.Info("server starting", zap.String("address", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	// ——— Graceful shutdown ———————————————————————————————————————————————————
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // block until signal received

	log.Info("shutdown signal received, draining connections...")

	// Give in-flight requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("graceful shutdown failed", zap.Error(err))
	}

	// Shutdown worker pool
	app.WorkerPool.Shutdown()

	// Close DB connection pool
	if err := app.DB.Close(); err != nil {
		log.Error("db close error", zap.Error(err))
	}

	// Close Redis client
	if err := app.Redis.Close(); err != nil {
		log.Error("redis close error", zap.Error(err))
	}

	log.Info("server exited cleanly")
}
