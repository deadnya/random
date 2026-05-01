package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

type rollEvent struct {
	UserID       int64  `json:"user_id"`
	Username     string `json:"username"`
	RolledNumber int    `json:"rolled_number"`
	TotalScore   int    `json:"total_score"`
}

func waitForTopic(brokers, topic string) {
	for {
		conn, err := kafka.Dial("tcp", brokers)
		if err != nil {
			log.Printf("consumer: waiting for kafka...")
			time.Sleep(2 * time.Second)
			continue
		}

		partitions, err := conn.ReadPartitions()
		conn.Close()
		if err != nil {
			log.Printf("consumer: waiting for kafka topics...")
			time.Sleep(2 * time.Second)
			continue
		}

		for _, p := range partitions {
			if p.Topic == topic {
				log.Printf("consumer: topic %q found", topic)
				return
			}
		}

		log.Printf("consumer: waiting for topic %q to be created...", topic)
		time.Sleep(2 * time.Second)
	}
}

func runConsumer(ctx context.Context, cfg config, agg *aggregator) {
	waitForTopic(cfg.KafkaBrokers, cfg.KafkaTopic)

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{cfg.KafkaBrokers},
		Topic:   cfg.KafkaTopic,
		GroupID: cfg.KafkaGroupID,
	})
	defer reader.Close()

	log.Printf("consumer started: brokers=%s topic=%s group=%s", cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroupID)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if strings.Contains(err.Error(), "Unknown Topic Or Partition") {
				log.Printf("consumer: topic not available yet, retrying...")
				time.Sleep(2 * time.Second)
				continue
			}
			log.Printf("consumer read error: %v", err)
			continue
		}

		var evt rollEvent
		if err := json.Unmarshal(msg.Value, &evt); err != nil {
			log.Printf("consumer unmarshal error: %v", err)
			continue
		}

		agg.ingest(evt.UserID, evt.Username, evt.RolledNumber, evt.TotalScore)
		log.Printf("consumer: ingested roll for user %d (score=%d)", evt.UserID, evt.TotalScore)
	}
}
