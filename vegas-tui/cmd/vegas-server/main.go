package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/choice404/vegas-protocol/vegas-tui/internal/config"
	"github.com/choice404/vegas-protocol/vegas-tui/internal/db"
	"github.com/choice404/vegas-protocol/vegas-tui/internal/server"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Database is optional — only connect if configured
	var pool *pgxpool.Pool
	if cfg.HasDatabase {
		pool, err = db.Connect(ctx, cfg.DatabaseURL)
		if err != nil {
			log.Printf("WARNING: db connection failed: %v (continuing without DB)", err)
		} else {
			defer pool.Close()
		}
	}

	router := server.NewRouter(cfg, pool)

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
		srv.Close()
	}()

	log.Printf("V.E.G.A.S. Server listening on :%s", cfg.ServerPort)
	if pool != nil {
		log.Println("Mode: Full (DB + Auth + Chat)")
	} else {
		log.Println("Mode: Chat-only (Ollama relay)")
	}
	log.Printf("Ollama: %s", cfg.OllamaURL)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}
