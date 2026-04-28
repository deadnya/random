package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type rollStatus struct {
	Available         int
	NextRollInSeconds int
}

func (s *server) currentRollStatus(ctx context.Context, userID int64) (rollStatus, error) {
	status, _, err := s.mutateRollState(ctx, userID, false)
	return status, err
}

func (s *server) consumeRoll(ctx context.Context, userID int64) (rollStatus, bool, error) {
	return s.mutateRollState(ctx, userID, true)
}

func (s *server) mutateRollState(ctx context.Context, userID int64, consume bool) (rollStatus, bool, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return rollStatus{}, false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var available int
	var lastRefillAt time.Time
	err = tx.QueryRow(ctx, `
		SELECT rolls_available, last_refill_at
		FROM user_roll_state
		WHERE user_id = $1
		FOR UPDATE
	`, userID).Scan(&available, &lastRefillAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			available = s.cfg.RollMaxTokens
			lastRefillAt = time.Now().UTC()
			_, err = tx.Exec(ctx, `
				INSERT INTO user_roll_state (user_id, rolls_available, last_refill_at, updated_at)
				VALUES ($1, $2, $3, NOW())
			`, userID, available, lastRefillAt)
			if err != nil {
				return rollStatus{}, false, fmt.Errorf("create roll state: %w", err)
			}
		} else {
			return rollStatus{}, false, fmt.Errorf("select roll state: %w", err)
		}
	}

	now := time.Now().UTC()
	available, lastRefillAt = refillRolls(available, lastRefillAt, now, s.cfg.RollMaxTokens, s.cfg.RollRefillSeconds)

	consumed := false
	if consume && available > 0 {
		available--
		consumed = true
	}

	_, err = tx.Exec(ctx, `
		UPDATE user_roll_state
		SET rolls_available = $2, last_refill_at = $3, updated_at = NOW()
		WHERE user_id = $1
	`, userID, available, lastRefillAt)
	if err != nil {
		return rollStatus{}, false, fmt.Errorf("update roll state: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return rollStatus{}, false, fmt.Errorf("commit tx: %w", err)
	}

	return rollStatus{
		Available:         available,
		NextRollInSeconds: secondsUntilNextRoll(available, lastRefillAt, now, s.cfg.RollMaxTokens, s.cfg.RollRefillSeconds),
	}, consumed, nil
}

func refillRolls(available int, lastRefillAt, now time.Time, maxTokens, refillSeconds int) (int, time.Time) {
	if available >= maxTokens {
		return maxTokens, now
	}

	elapsed := now.Sub(lastRefillAt)
	refills := int(elapsed.Seconds()) / refillSeconds
	if refills <= 0 {
		return available, lastRefillAt
	}

	available += refills
	if available >= maxTokens {
		return maxTokens, now
	}

	advanced := time.Duration(refills*refillSeconds) * time.Second
	return available, lastRefillAt.Add(advanced)
}

func secondsUntilNextRoll(available int, lastRefillAt, now time.Time, maxTokens, refillSeconds int) int {
	if available >= maxTokens {
		return 0
	}

	elapsed := int(now.Sub(lastRefillAt).Seconds())
	remaining := refillSeconds - (elapsed % refillSeconds)
	if remaining <= 0 {
		return refillSeconds
	}
	return remaining
}
