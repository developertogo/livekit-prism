package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Config struct {
	Port             string
	LiveKitURL       string
	LiveKitAPIKey    string
	LiveKitAPISecret string
	ClickHouseDSN    string
	PostHogHost      string
	PostHogProject   string
	PostHogAPIKey    string
}

func loadConfig() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	lkURL := os.Getenv("LIVEKIT_URL")
	if lkURL == "" {
		lkURL = "http://127.0.0.1:7880"
	}

	lkKey := os.Getenv("LIVEKIT_API_KEY")
	if lkKey == "" {
		lkKey = "devkey"
	}

	lkSecret := os.Getenv("LIVEKIT_API_SECRET")
	if lkSecret == "" {
		lkSecret = "secret"
	}

	chDSN := os.Getenv("CLICKHOUSE_DSN")
	if chDSN == "" {
		chDSN = "clickhouse://default:@127.0.0.1:9000/livekit_telemetry"
	}

	phHost := os.Getenv("POSTHOG_HOST")
	if phHost == "" {
		phHost = "https://us.i.posthog.com"
	}

	return Config{
		Port:             port,
		LiveKitURL:       lkURL,
		LiveKitAPIKey:    lkKey,
		LiveKitAPISecret: lkSecret,
		ClickHouseDSN:    chDSN,
		PostHogHost:      phHost,
		PostHogProject:   os.Getenv("POSTHOG_PROJECT_ID"),
		PostHogAPIKey:    os.Getenv("POSTHOG_PERSONAL_API_KEY"),
	}
}

func main() {
	cfg := loadConfig()

	log.Printf("=====================================================")
	log.Printf(" Starting Prism Multimodal Tool Gateway (Go Core)")
	log.Printf(" LiveKit Host:     %s", cfg.LiveKitURL)
	log.Printf(" ClickHouse DSN:   %s", cfg.ClickHouseDSN)
	log.Printf(" PostHog Host:     %s", cfg.PostHogHost)
	log.Printf(" HTTP Listen Port: :%s", cfg.Port)
	log.Printf("=====================================================")

	// Initialize LiveKit gRPC wrapper
	lkClient := NewLiveKitClient(cfg.LiveKitURL, cfg.LiveKitAPIKey, cfg.LiveKitAPISecret)

	// Initialize ClickHouse telemetry batch recorder
	telemetry := NewTelemetryRecorder(cfg.ClickHouseDSN, 1024)
	defer telemetry.Close()

	// Initialize Dual-Channel Optical Router
	router := NewDualChannelRouter(
		lkClient,
		telemetry,
		cfg.PostHogHost,
		cfg.PostHogProject,
		cfg.PostHogAPIKey,
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","service":"prism-gateway"}`)
	})
	mux.HandleFunc("/api/tools/execute", router.HandleToolExecution)
	mux.HandleFunc("/api/tools/funnel", router.HandleToolExecution)
	mux.HandleFunc("/api/tools/hogql", router.HandleHogQL)
	mux.HandleFunc("/api/tools/filler", router.HandleFiller)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Server run loop
	go func() {
		log.Printf("[Prism Gateway] Listening on http://localhost:%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[Prism Gateway] Fatal server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("[Prism Gateway] Shutting down server gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("[Prism Gateway] Server shutdown forced: %v", err)
	}
	log.Printf("[Prism Gateway] Stopped.")
}
