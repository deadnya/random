package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type httpServer struct {
	agg *aggregator
}

func (s *httpServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /leaderboard/best-score", s.handleBestScore)
	mux.HandleFunc("GET /leaderboard/total-value", s.handleTotalValue)
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	return mux
}

func (s *httpServer) handleBestScore(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 10)
	entries := s.agg.getBestScore(limit)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"entries": entries})
}

func (s *httpServer) handleTotalValue(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 10)
	entries := s.agg.getTotalValue(limit)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"entries": entries})
}

func (s *httpServer) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func parseLimit(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}

func runRefresher(ctx context.Context, interval time.Duration, agg *aggregator, db *pgxpool.Pool) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			agg.refresh()
			if db != nil {
				if err := agg.saveToDB(ctx, db); err != nil {
					log.Printf("failed to save aggregates to db: %v", err)
				} else {
					log.Printf("refreshed leaderboards and saved aggregates")
				}
			} else {
				log.Printf("refreshed leaderboards")
			}
		}
	}
}
