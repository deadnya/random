package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
)

type outboxWorker struct {
	db     *pgxpool.Pool
	writer *kafka.Writer
	batch  int
}

func newOutboxWorker(db *pgxpool.Pool, writer *kafka.Writer) *outboxWorker {
	return &outboxWorker{
		db:     db,
		writer: writer,
		batch:  100,
	}
}

func (w *outboxWorker) run(ctx context.Context) {
	if w.db == nil || w.writer == nil {
		return
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				log.Printf("outbox: batch error: %v", err)
			}
		}
	}
}

func (w *outboxWorker) processBatch(ctx context.Context) error {
	tx, err := w.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT id, topic, key, payload
		FROM outbox
		WHERE processed_at IS NULL
		ORDER BY id
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, w.batch)
	if err != nil {
		return fmt.Errorf("query outbox: %w", err)
	}
	defer rows.Close()

	type item struct {
		id      int64
		topic   string
		key     string
		payload []byte
	}

	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.topic, &it.key, &it.payload); err != nil {
			return fmt.Errorf("scan outbox: %w", err)
		}
		items = append(items, it)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate outbox: %w", err)
	}

	if len(items) == 0 {
		return nil
	}

	msgs := make([]kafka.Message, 0, len(items))
	ids := make([]int64, 0, len(items))
	for _, it := range items {
		msgs = append(msgs, kafka.Message{
			Key:   []byte(it.key),
			Value: it.payload,
		})
		ids = append(ids, it.id)
	}

	if err := w.writer.WriteMessages(ctx, msgs...); err != nil {
		return fmt.Errorf("publish to kafka: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE outbox
		SET processed_at = NOW()
		WHERE id = ANY($1)
	`, ids)
	if err != nil {
		return fmt.Errorf("mark processed: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	log.Printf("outbox: published %d messages", len(items))
	return nil
}
