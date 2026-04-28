package main

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"numbers/internal/ui"
)

type server struct {
	cfg    config
	db     *pgxpool.Pool
	tmpl   *template.Template
	scorer *rarityScorer
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/roll", s.handleRoll)
	mux.HandleFunc("/roll/state", s.handleRollState)
	mux.HandleFunc("/profile/init", s.handleProfileInit)
	mux.HandleFunc("/profile/view", s.handleProfileView)
	mux.HandleFunc("/profile/username", s.handleProfileUsername)
	mux.HandleFunc("/history", s.handleHistory)
	mux.HandleFunc("/specs/unlocked", s.handleUnlockedSpecs)
	mux.HandleFunc("/leaderboard", s.handleLeaderboard)
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/leaderboard/total-value", s.handleTotalValueLeaderboard)
	return mux
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data := struct {
		RollMaxTokens     int
		RollRefillSeconds int
	}{
		RollMaxTokens:     s.cfg.RollMaxTokens,
		RollRefillSeconds: s.cfg.RollRefillSeconds,
	}

	if err := s.tmpl.Execute(w, data); err != nil {
		http.Error(w, "template render failed", http.StatusInternalServerError)
	}
}

func (s *server) handleRoll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := s.userFromRequest(r)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(ui.RenderNeedsProfileFragment()))
		return
	}

	status, consumed, err := s.consumeRoll(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "roll state unavailable", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "refresh-roll-state")
	if !consumed {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(ui.RenderNoRollsFragment(status.NextRollInSeconds)))
		return
	}

	number, err := secureIntn(1_000_000)
	if err != nil {
		http.Error(w, "random generation failed", http.StatusInternalServerError)
		return
	}

	if s.scorer == nil {
		http.Error(w, "scorer unavailable", http.StatusInternalServerError)
		return
	}

	specs, total := s.scorer.calculate(number)
	if err := s.persistRoll(r.Context(), user.ID, number, total, specs); err != nil {
		http.Error(w, "unable to save roll", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("HX-Trigger", `{"refresh-roll-state":true,"refresh-history":true,"refresh-unlocked-specs":true,"refresh-leaderboard":true}`)
	_, _ = w.Write([]byte(ui.RenderRollFragment(number, toUISpecs(specs), total)))
}

func (s *server) handleRollState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := s.userFromRequest(r)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(ui.RenderNeedsProfileFragment()))
		return
	}

	status, err := s.currentRollStatus(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "roll state unavailable", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(ui.RenderRollControls(ui.RollStatus{Available: status.Available, NextRollInSeconds: status.NextRollInSeconds}, s.cfg.RollMaxTokens)))
}

func (s *server) handleProfileInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	profileID, err := newProfileID()
	if err != nil {
		http.Error(w, "unable to create profile", http.StatusInternalServerError)
		return
	}

	user, err := s.ensureUserByProfileID(r.Context(), profileID)
	if err != nil {
		http.Error(w, "unable to initialize profile", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "numbers_profile_id",
		Value:    url.QueryEscape(user.PublicID),
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 365,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"profile_id": user.PublicID,
		"username":   user.Username,
	})
}

func (s *server) handleProfileView(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := s.userFromRequest(r)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(ui.RenderPanelMessage("Profile", "Profile is still initializing. Please wait a moment.")))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(ui.RenderProfilePanel(ui.ProfileEntry{PublicID: user.PublicID, Username: user.Username})))
}

func (s *server) handleProfileUsername(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := s.userFromRequest(r)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(ui.RenderPanelMessage("Profile", "Profile is still initializing. Please wait a moment.")))
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	if len(username) < 3 || len(username) > 24 {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(ui.RenderProfilePanelWithMessage(ui.ProfileEntry{PublicID: user.PublicID, Username: user.Username}, "Username must be 3-24 characters.")))
		return
	}

	updated, err := s.updateUsername(r.Context(), user.ID, username)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(ui.RenderProfilePanelWithMessage(ui.ProfileEntry{PublicID: user.PublicID, Username: user.Username}, "Unable to update username right now.")))
		return
	}

	w.Header().Set("HX-Trigger", `{"refresh-leaderboard":true}`)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(ui.RenderProfilePanelWithMessage(ui.ProfileEntry{PublicID: updated.PublicID, Username: updated.Username}, "Username updated.")))
}

func (s *server) handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := s.userFromRequest(r)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(ui.RenderPanelMessage("Roll History", "Profile is still initializing. Please wait a moment.")))
		return
	}

	history, err := s.fetchRollHistory(r.Context(), user.ID, 10)
	if err != nil {
		http.Error(w, "unable to load history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	items := make([]ui.HistoryEntry, 0, len(history))
	for _, item := range history {
		items = append(items, ui.HistoryEntry{Number: item.Number, Score: item.Score, CreatedAt: item.CreatedAt})
	}
	_, _ = w.Write([]byte(ui.RenderHistoryPanel(items)))
}

func (s *server) handleUnlockedSpecs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := s.userFromRequest(r)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(ui.RenderPanelMessage("Unlocked Specs", "Profile is still initializing. Please wait a moment.")))
		return
	}

	items, err := s.fetchUnlockedSpecs(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "unable to load unlocked specs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	converted := make([]ui.UnlockedSpecEntry, 0, len(items))
	for _, item := range items {
		converted = append(converted, ui.UnlockedSpecEntry{SpecKey: item.SpecKey, RollCount: item.RollCount})
	}
	_, _ = w.Write([]byte(ui.RenderUnlockedSpecsPanel(converted)))
}

func (s *server) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := s.fetchLeaderboard(r.Context(), 10)
	if err != nil {
		http.Error(w, "unable to load leaderboard", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	converted := make([]ui.LeaderboardEntry, 0, len(rows))
	for _, row := range rows {
		converted = append(converted, ui.LeaderboardEntry{Username: row.Username, BestScore: row.BestScore, RollCount: row.RollCount, BestNumber: row.BestNumber})
	}
	_, _ = w.Write([]byte(ui.RenderLeaderboardPanel(converted)))
}

func (s *server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := s.db.Ping(ctx); err != nil {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *server) handleTotalValueLeaderboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	items, err := s.fetchTotalValueLeaderboard(ctx, 10)
	if err != nil {
		http.Error(w, "failed to load leaderboard", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	converted := make([]ui.TotalValueLeaderboardEntry, 0, len(items))
	for _, item := range items {
		converted = append(converted, ui.TotalValueLeaderboardEntry{Username: item.Username, TotalValue: item.TotalValue, RollCount: item.RollCount, BestNumber: item.BestNumber})
	}
	w.Write([]byte(ui.RenderTotalValueLeaderboardPanel(converted)))
}