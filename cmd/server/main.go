package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"
)

func main() {
	cfg := loadConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := newDBPool(ctx, cfg)
	if err != nil {
		log.Fatalf("unable to connect to database: %v", err)
	}
	defer db.Close()

	scorer, err := loadRarityScorer(ctx, db, cfg.RarityScoreScale)
	if err != nil {
		log.Fatalf("unable to load rarity scorer: %v", err)
	}

	tmpl, err := template.ParseFiles("web/templates/index.html")
	if err != nil {
		log.Fatalf("unable to parse templates: %v", err)
	}

	var producer *kafkaProducer
	if cfg.KafkaBrokers != "" && cfg.KafkaTopic != "" {
		if err := ensureKafkaTopic(cfg.KafkaBrokers, cfg.KafkaTopic); err != nil {
			log.Printf("kafka: unable to ensure topic: %v", err)
		}
		producer = newKafkaProducer(cfg.KafkaBrokers, cfg.KafkaTopic)
		defer producer.close()

		worker := newOutboxWorker(db, producer.writer)
		go worker.run(context.Background())
	}

	var lbClient *leaderboardClient
	if cfg.LeaderboardServiceURL != "" {
		lbClient = newLeaderboardClient(cfg.LeaderboardServiceURL)
	}

	srv := &server{cfg: cfg, db: db, tmpl: tmpl, scorer: scorer, kafkaProducer: producer, leaderboardClient: lbClient}
	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.AppPort),
		Handler:           logRequest(srv.routes()),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("numbers server running on :%d with %d rarity specs", cfg.AppPort, len(scorer.odds))
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
