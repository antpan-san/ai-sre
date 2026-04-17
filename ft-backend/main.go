package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ft-backend/common/config"
	"ft-backend/common/logger"
	"ft-backend/common/redis"
	"ft-backend/database"
	"ft-backend/iotservice"
	"ft-backend/routes"
	"ft-backend/utils"
)

func main() {
	const configPath = "conf/config.yaml"

	// 1. Ensure config file exists
	if err := config.EnsureConfigExists(configPath); err != nil {
		logger.Error("Failed to ensure config exists: %v", err)
		return
	}

	// 2. Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("Failed to load config: %v", err)
		return
	}
	config.GlobalCfg = cfg
	logger.InitLogger(cfg.Log.Level, nil)
	logger.Info("Configuration loaded successfully")

	// 3. Connect to PostgreSQL
	if err := database.Connect(&cfg.Database); err != nil {
		logger.Error("Failed to connect to database: %v", err)
		return
	}
	defer database.Close()

	// 4. Run database migrations
	if err := database.Migrate(); err != nil {
		logger.Error("Failed to migrate database: %v", err)
		return
	}

	// 5. Connect to Redis (non-fatal if fails — graceful degradation)
	if err := redis.Connect(&cfg.Redis); err != nil {
		logger.Warn("Redis not available, running without cache/queue: %v", err)
	} else {
		defer redis.Close()
	}

	// 6. Create upload directory
	if err := os.MkdirAll(cfg.File.UploadDir, 0755); err != nil {
		logger.Error("Failed to create upload directory: %v", err)
		return
	}

	// 7. Start WebSocket manager
	utils.GlobalWebSocketManager = utils.NewWebSocketManager()
	go utils.GlobalWebSocketManager.Start()

	// 8. Start machine status monitor
	go utils.StartMachineStatusMonitor()

	// 9. Start heartbeat consumer (only if Redis is available)
	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	defer consumerCancel()
	if redis.IsConnected() {
		go iotservice.StartHeartbeatConsumer(consumerCtx)
	}

	// 10. Setup router
	router := routes.SetupRouter(cfg)

	// 11. Create HTTP server with timeouts
	serverAddr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 12. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("Server starting on %s", serverAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server listen error: %v", err)
			quit <- syscall.SIGTERM
		}
	}()

	// Block until signal
	sig := <-quit
	logger.Info("Received signal %v, shutting down gracefully...", sig)

	// Give outstanding requests 15 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced shutdown: %v", err)
	}

	logger.Info("Server exited cleanly")
}
