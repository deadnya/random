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

func main() {
	cfg := loadConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := newDBPool(ctx, cfg)
	if err != nil {
		log.Fatalf("unable to connect to database: %v", err)
	}
	defer db.Close()

	agg := newAggregator()
	if err := agg.loadFromDB(ctx, db); err != nil {
		log.Printf("warning: failed to load aggregates from db: %v", err)
	}
	agg.refresh()

	go runConsumer(ctx, cfg, agg)
	go runRefresher(ctx, time.Duration(cfg.LeaderboardRefreshIntervalSec)*time.Second, agg, db)

	srv := &httpServer{agg: agg}
	httpSrv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.AppPort),
		Handler:           srv.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down leaderboard service...")
		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	}()

	log.Printf("leaderboard service running on :%d", cfg.AppPort)
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("leaderboard service stopped: %v", err)
	}
}
