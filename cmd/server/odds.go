package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func loadRarityScorer(ctx context.Context, db *pgxpool.Pool, scale float64) (*rarityScorer, error) {
	if scale <= 0 {
		scale = 150
	}

	requiredKeys := requiredOddsKeys()

	odds, err := fetchSpecOdds(ctx, db, scale, requiredKeys)
	if err != nil {
		return nil, err
	}

	missing := missingOddsKeys(odds, requiredKeys)
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing odds rows for keys: %s", strings.Join(missing, ", "))
	}

	return &rarityScorer{odds: odds}, nil
}

func fetchSpecOdds(ctx context.Context, db *pgxpool.Pool, scale float64, requiredKeys []string) (map[string]specOdd, error) {
	rows, err := db.Query(ctx, `SELECT spec_key, probability FROM spec_odds`)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42P01" {
			return nil, fmt.Errorf("query spec odds: spec_odds table not found; run migrations first")
		}
		return nil, fmt.Errorf("query spec odds: %w", err)
	}
	defer rows.Close()

	odds := make(map[string]specOdd, len(requiredKeys))
	for rows.Next() {
		var key string
		var probability float64
		if err := rows.Scan(&key, &probability); err != nil {
			return nil, fmt.Errorf("scan spec odds: %w", err)
		}

		if probability <= 0 || probability > 1 {
			return nil, fmt.Errorf("invalid probability for key %q: %f", key, probability)
		}

		odds[key] = specOdd{
			Probability: probability,
			Score:       rarityScore(probability, scale),
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate spec odds: %w", err)
	}

	if len(odds) == 0 {
		return nil, errors.New("spec_odds table is empty")
	}

	return odds, nil
}

func rarityScore(probability, scale float64) int {
	if probability <= 0 {
		return 0
	}

	if probability > 1 {
		probability = 1
	}

	score := int(math.Round(scale * math.Log10(1/probability)))
	if score < 1 {
		return 1
	}
	return score
}

func requiredOddsKeys() []string {
	keys := make([]string, 0, len(specRules))
	seen := make(map[string]struct{}, len(specRules))

	for _, rule := range specRules {
		if _, exists := seen[rule.Key]; exists {
			continue
		}
		seen[rule.Key] = struct{}{}
		keys = append(keys, rule.Key)
	}

	return keys
}

func missingOddsKeys(odds map[string]specOdd, requiredKeys []string) []string {
	missing := make([]string, 0)
	for _, key := range requiredKeys {
		if _, ok := odds[key]; !ok {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	return missing
}
