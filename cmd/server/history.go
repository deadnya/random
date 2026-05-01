package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type rollHistoryEntry struct {
	Number    int
	Score     int
	CreatedAt time.Time
}

type unlockedSpecEntry struct {
	SpecKey   string
	RollCount int
}

type totalValueLeaderboardEntry struct {
	Username   string
	TotalValue int
	RollCount  int
	BestNumber int
}

type leaderboardEntry struct {
	Username   string
	BestScore  int
	RollCount  int
	BestNumber int
}

func (s *server) persistRoll(ctx context.Context, userID int64, username string, number, total int, specs []spec) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var rollID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO rolls (user_id, rolled_number, total_score)
		VALUES ($1, $2, $3)
		RETURNING id
	`, userID, number, total).Scan(&rollID)
	if err != nil {
		return fmt.Errorf("insert roll: %w", err)
	}

	for _, item := range specs {
		_, err := tx.Exec(ctx, `
			INSERT INTO roll_specs (roll_id, spec_key, spec_value, spec_score)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (roll_id, spec_key) DO UPDATE
			SET spec_value = EXCLUDED.spec_value,
			    spec_score = EXCLUDED.spec_score
		`, rollID, item.Key, item.Value, item.Score)
		if err != nil {
			return fmt.Errorf("insert roll spec: %w", err)
		}
	}

	if s.cfg.KafkaTopic != "" {
		evt := rollEvent{
			UserID:       userID,
			Username:     username,
			RolledNumber: number,
			TotalScore:   total,
			CreatedAt:    time.Now().UTC(),
		}
		data, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("marshal outbox payload: %w", err)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO outbox (topic, key, payload)
			VALUES ($1, $2, $3)
		`, s.cfg.KafkaTopic, fmt.Sprintf("%d", userID), data)
		if err != nil {
			return fmt.Errorf("insert outbox: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (s *server) fetchRollHistory(ctx context.Context, userID int64, limit int) ([]rollHistoryEntry, error) {
	rows, err := s.db.Query(ctx, `
		SELECT rolled_number, total_score, created_at
		FROM rolls
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("query roll history: %w", err)
	}
	defer rows.Close()

	items := make([]rollHistoryEntry, 0, limit)
	for rows.Next() {
		var item rollHistoryEntry
		if err := rows.Scan(&item.Number, &item.Score, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan roll history: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate roll history: %w", err)
	}

	return items, nil
}

func (s *server) fetchUnlockedSpecs(ctx context.Context, userID int64) ([]unlockedSpecEntry, error) {
	rows, err := s.db.Query(ctx, `
		SELECT rs.spec_key,
		       COUNT(*) AS roll_count
		FROM roll_specs rs
		INNER JOIN rolls r ON r.id = rs.roll_id
		LEFT JOIN spec_odds so ON so.spec_key = rs.spec_key
		WHERE r.user_id = $1
		GROUP BY rs.spec_key, so.probability
		ORDER BY COALESCE(so.probability, 1.0) ASC,
		         roll_count DESC,
		         rs.spec_key ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query unlocked specs: %w", err)
	}
	defer rows.Close()

	items := make([]unlockedSpecEntry, 0)
	for rows.Next() {
		var item unlockedSpecEntry
		if err := rows.Scan(&item.SpecKey, &item.RollCount); err != nil {
			return nil, fmt.Errorf("scan unlocked specs: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate unlocked specs: %w", err)
	}

	return items, nil
}

func (s *server) fetchLeaderboard(ctx context.Context, limit int) ([]leaderboardEntry, error) {
	if s.leaderboardClient != nil {
		entries, err := s.leaderboardClient.fetchLeaderboard(ctx, limit)
		if err == nil {
			return entries, nil
		}
	}

	rows, err := s.db.Query(ctx, `
		SELECT u.username,
		       MAX(r.total_score) AS best_score,
		       COUNT(r.id) AS roll_count,
		       (SELECT r2.rolled_number
		        FROM rolls r2
		        WHERE r2.user_id = u.id
		        ORDER BY r2.total_score DESC, r2.created_at DESC
		        LIMIT 1) AS best_number
		FROM users u
		INNER JOIN rolls r ON r.user_id = u.id
		GROUP BY u.id, u.username
		ORDER BY best_score DESC, roll_count DESC, u.username ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query leaderboard: %w", err)
	}
	defer rows.Close()

	items := make([]leaderboardEntry, 0, limit)
	for rows.Next() {
		var item leaderboardEntry
		if err := rows.Scan(&item.Username, &item.BestScore, &item.RollCount, &item.BestNumber); err != nil {
			return nil, fmt.Errorf("scan leaderboard: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate leaderboard: %w", err)
	}

	return items, nil
}

func (s *server) fetchTotalValueLeaderboard(ctx context.Context, limit int) ([]totalValueLeaderboardEntry, error) {
	if s.leaderboardClient != nil {
		entries, err := s.leaderboardClient.fetchTotalValueLeaderboard(ctx, limit)
		if err == nil {
			return entries, nil
		}
	}

	rows, err := s.db.Query(ctx, `
		SELECT u.username,
		       SUM(r.total_score) AS total_value,
		       COUNT(r.id) AS roll_count,
		       (SELECT r2.rolled_number
		        FROM rolls r2
		        WHERE r2.user_id = u.id
		        ORDER BY r2.total_score DESC, r2.created_at DESC
		        LIMIT 1) AS best_number
		FROM users u
		INNER JOIN rolls r ON r.user_id = u.id
		GROUP BY u.id, u.username
		ORDER BY total_value DESC, roll_count DESC, u.username ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query total-value leaderboard: %w", err)
	}
	defer rows.Close()

	items := make([]totalValueLeaderboardEntry, 0, limit)
	for rows.Next() {
		var item totalValueLeaderboardEntry
		if err := rows.Scan(&item.Username, &item.TotalValue, &item.RollCount, &item.BestNumber); err != nil {
			return nil, fmt.Errorf("scan total-value leaderboard: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate total-value leaderboard: %w", err)
	}

	return items, nil
}