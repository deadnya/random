package main

import (
	"context"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type userAggregate struct {
	UserID     int64
	Username   string
	BestScore  int
	TotalValue int
	RollCount  int
	BestNumber int
}

type aggregator struct {
	mu          sync.RWMutex
	users       map[int64]*userAggregate
	bestScore   []leaderboardEntry
	totalValue  []totalValueLeaderboardEntry
	refreshedAt time.Time
}

type leaderboardEntry struct {
	Username   string `json:"username"`
	BestScore  int    `json:"best_score"`
	RollCount  int    `json:"roll_count"`
	BestNumber int    `json:"best_number"`
}

type totalValueLeaderboardEntry struct {
	Username   string `json:"username"`
	TotalValue int    `json:"total_value"`
	RollCount  int    `json:"roll_count"`
	BestNumber int    `json:"best_number"`
}

func newAggregator() *aggregator {
	return &aggregator{
		users: make(map[int64]*userAggregate),
	}
}

func (a *aggregator) ingest(userID int64, username string, rolledNumber, totalScore int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	u, ok := a.users[userID]
	if !ok {
		u = &userAggregate{
			UserID:   userID,
			Username: username,
		}
		a.users[userID] = u
	}

	if username != "" {
		u.Username = username
	}
	u.RollCount++
	u.TotalValue += totalScore
	if totalScore > u.BestScore {
		u.BestScore = totalScore
		u.BestNumber = rolledNumber
	}
}

func (a *aggregator) refresh() {
	a.mu.Lock()
	defer a.mu.Unlock()

	best := make([]leaderboardEntry, 0, len(a.users))
	total := make([]totalValueLeaderboardEntry, 0, len(a.users))

	for _, u := range a.users {
		best = append(best, leaderboardEntry{
			Username:   u.Username,
			BestScore:  u.BestScore,
			RollCount:  u.RollCount,
			BestNumber: u.BestNumber,
		})
		total = append(total, totalValueLeaderboardEntry{
			Username:   u.Username,
			TotalValue: u.TotalValue,
			RollCount:  u.RollCount,
			BestNumber: u.BestNumber,
		})
	}

	sort.Slice(best, func(i, j int) bool {
		if best[i].BestScore != best[j].BestScore {
			return best[i].BestScore > best[j].BestScore
		}
		if best[i].RollCount != best[j].RollCount {
			return best[i].RollCount > best[j].RollCount
		}
		return best[i].Username < best[j].Username
	})

	sort.Slice(total, func(i, j int) bool {
		if total[i].TotalValue != total[j].TotalValue {
			return total[i].TotalValue > total[j].TotalValue
		}
		if total[i].RollCount != total[j].RollCount {
			return total[i].RollCount > total[j].RollCount
		}
		return total[i].Username < total[j].Username
	})

	a.bestScore = best
	a.totalValue = total
	a.refreshedAt = time.Now().UTC()
}

func (a *aggregator) getBestScore(limit int) []leaderboardEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if limit > len(a.bestScore) {
		limit = len(a.bestScore)
	}
	out := make([]leaderboardEntry, limit)
	copy(out, a.bestScore[:limit])
	return out
}

func (a *aggregator) getTotalValue(limit int) []totalValueLeaderboardEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if limit > len(a.totalValue) {
		limit = len(a.totalValue)
	}
	out := make([]totalValueLeaderboardEntry, limit)
	copy(out, a.totalValue[:limit])
	return out
}

func (a *aggregator) loadFromDB(ctx context.Context, db *pgxpool.Pool) error {
	rows, err := db.Query(ctx, `
		SELECT user_id, username, best_score, total_value, roll_count, best_number
		FROM user_aggregates
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	a.mu.Lock()
	defer a.mu.Unlock()

	count := 0
	for rows.Next() {
		var u userAggregate
		if err := rows.Scan(&u.UserID, &u.Username, &u.BestScore, &u.TotalValue, &u.RollCount, &u.BestNumber); err != nil {
			return err
		}
		a.users[u.UserID] = &u
		count++
	}

	if err := rows.Err(); err != nil {
		return err
	}

	log.Printf("loaded %d user aggregates from db", count)
	return nil
}

func (a *aggregator) saveToDB(ctx context.Context, db *pgxpool.Pool) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.users) == 0 {
		return nil
	}

	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM user_aggregates`); err != nil {
		return err
	}

	batch := &pgx.Batch{}
	for _, u := range a.users {
		batch.Queue(`
			INSERT INTO user_aggregates (user_id, username, best_score, total_value, roll_count, best_number, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW())
		`, u.UserID, u.Username, u.BestScore, u.TotalValue, u.RollCount, u.BestNumber)
	}

	br := tx.SendBatch(ctx, batch)
	if err := br.Close(); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
