package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type leaderboardClient struct {
	baseURL string
	client  *http.Client
}

func newLeaderboardClient(baseURL string) *leaderboardClient {
	return &leaderboardClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

type leaderboardServiceEntry struct {
	Username   string `json:"username"`
	BestScore  int    `json:"best_score"`
	RollCount  int    `json:"roll_count"`
	BestNumber int    `json:"best_number"`
}

type bestScoreResponse struct {
	Entries []leaderboardServiceEntry `json:"entries"`
}

type totalValueServiceEntry struct {
	Username   string `json:"username"`
	TotalValue int    `json:"total_value"`
	RollCount  int    `json:"roll_count"`
	BestNumber int    `json:"best_number"`
}

type totalValueResponse struct {
	Entries []totalValueServiceEntry `json:"entries"`
}

func (c *leaderboardClient) fetchLeaderboard(ctx context.Context, limit int) ([]leaderboardEntry, error) {
	url := fmt.Sprintf("%s/leaderboard/best-score?limit=%d", c.baseURL, limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("leaderboard service returned %d", resp.StatusCode)
	}

	var payload bestScoreResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	entries := make([]leaderboardEntry, 0, len(payload.Entries))
	for _, e := range payload.Entries {
		entries = append(entries, leaderboardEntry{
			Username:   e.Username,
			BestScore:  e.BestScore,
			RollCount:  e.RollCount,
			BestNumber: e.BestNumber,
		})
	}
	return entries, nil
}

func (c *leaderboardClient) fetchTotalValueLeaderboard(ctx context.Context, limit int) ([]totalValueLeaderboardEntry, error) {
	url := fmt.Sprintf("%s/leaderboard/total-value?limit=%d", c.baseURL, limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("leaderboard service returned %d", resp.StatusCode)
	}

	var payload totalValueResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	entries := make([]totalValueLeaderboardEntry, 0, len(payload.Entries))
	for _, e := range payload.Entries {
		entries = append(entries, totalValueLeaderboardEntry{
			Username:   e.Username,
			TotalValue: e.TotalValue,
			RollCount:  e.RollCount,
			BestNumber: e.BestNumber,
		})
	}
	return entries, nil
}
