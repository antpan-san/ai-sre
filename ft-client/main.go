// OpsFleetPilot Client Agent
//
// A lightweight agent deployed in customer's intranet environment.
// It periodically sends heartbeats to the Server, receives commands,
// and executes operations via Ansible on managed nodes.
//
// Supports master/worker topology: each node reports its role and cluster
// membership, enabling the server to build a master->workers tree view.
//
// Usage:
//
//	./ft-client                                        # Use default config (conf/client.yaml)
//	./ft-client -config /path/to/config                # Specify config file
//	./ft-client -server https://x.x.x.x               # Override server URL
//	./ft-client -id my-client-01                       # Override client ID
//	./ft-client -role master -cluster-id prod-cluster  # Set topology role
//	./ft-client -token <auth-token>                    # Set auth token
//	./ft-client -version                               # Show version and exit
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"ft-client/internal/config"
	"ft-client/internal/heartbeat"
	"ft-client/internal/logger"
	"ft-client/internal/transport"
)

var (
	// Set via ldflags at build time:
	//   go build -ldflags "-X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
	buildTime = "unknown"
)

func main() {
	// -------------------------------------------------------------------------
	// 1. Parse command-line flags
	// -------------------------------------------------------------------------
	configFile := flag.String("config", "conf/client.yaml", "Path to config file")
	showVersion := flag.Bool("version", false, "Show version and exit")
	serverURL := flag.String("server", "", "Server URL (overrides config)")
	clientID := flag.String("id", "", "Client ID (overrides config)")
	role := flag.String("role", "", "Node role: master or worker (overrides config)")
	clusterID := flag.String("cluster-id", "", "Cluster ID (overrides config)")
	clusterName := flag.String("cluster-name", "", "Cluster display name (overrides config)")
	authToken := flag.String("token", "", "Auth token for server communication (overrides config)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("ft-client version %s (built: %s)\n", "1.0.0", buildTime)
		os.Exit(0)
	}

	// -------------------------------------------------------------------------
	// 2. Load configuration
	// -------------------------------------------------------------------------
	cfg, err := config.Load(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[WARN] Failed to load config %s: %v, using defaults\n", *configFile, err)
		cfg = config.Default()
	}

	// CLI overrides take precedence
	if *serverURL != "" {
		cfg.Server.URL = *serverURL
	}
	if *clientID != "" {
		cfg.Client.ID = *clientID
	}
	if *role != "" {
		cfg.Client.Role = *role
	}
	if *clusterID != "" {
		cfg.Cluster.ID = *clusterID
	}
	if *clusterName != "" {
		cfg.Cluster.Name = *clusterName
	}
	if *authToken != "" {
		cfg.Auth.Token = *authToken
	}

	// Track whether the client ID was freshly generated so we can persist it below.
	generatedNewID := cfg.Client.ID == ""
	if generatedNewID {
		cfg.Client.ID = config.GenerateClientID()
	}

	// Default cluster name to cluster ID if not set
	if cfg.Cluster.Name == "" && cfg.Cluster.ID != "" {
		cfg.Cluster.Name = cfg.Cluster.ID
	}

	// Persist newly-generated client_id back to the config file so it remains
	// stable across restarts.  Without this, every restart produces a new random
	// ID and any pending SubTasks that were queued for the old ID are never
	// consumed, breaking the sync_nodes delivery chain.
	if generatedNewID {
		if err := config.SaveTo(cfg, *configFile); err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] could not persist generated client_id to %s: %v\n", *configFile, err)
		}
	}

	// -------------------------------------------------------------------------
	// 3. Initialize logger
	// -------------------------------------------------------------------------
	if err := logger.Init(cfg.Log.Level, cfg.Log.File); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	logger.Info("OpsFleetPilot Client Agent starting",
		"version", cfg.Client.Version,
		"build_time", buildTime,
		"client_id", cfg.Client.ID,
		"server_url", cfg.Server.URL,
		"role", cfg.Client.Role,
		"cluster_id", cfg.Cluster.ID,
		"cluster_name", cfg.Cluster.Name,
		"heartbeat_interval", cfg.Heartbeat.Interval,
		"auth_enabled", cfg.Auth.Token != "",
	)

	// -------------------------------------------------------------------------
	// 4. Create HTTP transport (with TLS support)
	// -------------------------------------------------------------------------
	httpClient, err := transport.NewHTTPClient(cfg)
	if err != nil {
		logger.Error("failed to create HTTP client", "error", err)
		os.Exit(1)
	}

	// -------------------------------------------------------------------------
	// 5. Create command handler (Phase 1: stub, Phase 2: shell/ansible)
	// -------------------------------------------------------------------------
	cmdHandler := &heartbeat.StubHandler{
		ClientID: cfg.Client.ID,
	}

	// -------------------------------------------------------------------------
	// 6. Create heartbeat service
	// -------------------------------------------------------------------------
	hbService := heartbeat.NewService(cfg, httpClient, cmdHandler)

	// -------------------------------------------------------------------------
	// 7. Setup context with signal-based cancellation for graceful shutdown
	// -------------------------------------------------------------------------
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for SIGINT (Ctrl+C) and SIGTERM (kill)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info("received shutdown signal", "signal", sig.String())
		cancel()
	}()

	// -------------------------------------------------------------------------
	// 8. Start the heartbeat loop (blocks until context is cancelled)
	// -------------------------------------------------------------------------
	if err := hbService.Run(ctx); err != nil {
		logger.Error("heartbeat service exited with error", "error", err)
		os.Exit(1)
	}

	logger.Info("OpsFleetPilot Client Agent stopped gracefully")
}
