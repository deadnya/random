package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type rollEvent struct {
	UserID       int64     `json:"user_id"`
	Username     string    `json:"username"`
	RolledNumber int       `json:"rolled_number"`
	TotalScore   int       `json:"total_score"`
	CreatedAt    time.Time `json:"created_at"`
}

type kafkaProducer struct {
	writer *kafka.Writer
}

func ensureKafkaTopic(brokers, topic string) error {
	conn, err := kafka.Dial("tcp", brokers)
	if err != nil {
		return fmt.Errorf("dial kafka: %w", err)
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions()
	if err != nil {
		return fmt.Errorf("read partitions: %w", err)
	}

	for _, p := range partitions {
		if p.Topic == topic {
			log.Printf("kafka: topic %q already exists", topic)
			return nil
		}
	}

	err = conn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	if err != nil {
		return fmt.Errorf("create topic: %w", err)
	}

	log.Printf("kafka: created topic %q", topic)
	return nil
}

func newKafkaProducer(brokers, topic string) *kafkaProducer {
	return &kafkaProducer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireOne,
		},
	}
}

func (p *kafkaProducer) publishRollEvent(ctx context.Context, evt rollEvent) {
	if p == nil || p.writer == nil {
		return
	}

	data, err := json.Marshal(evt)
	if err != nil {
		log.Printf("kafka: failed to marshal roll event: %v", err)
		return
	}

	msg := kafka.Message{
		Key:   []byte(fmt.Sprintf("%d", evt.UserID)),
		Value: data,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		log.Printf("kafka: failed to write roll event: %v", err)
	} else {
		log.Printf("kafka: published roll event for user %d", evt.UserID)
	}
}

func (p *kafkaProducer) close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}
