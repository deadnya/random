package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"
)

type userProfile struct {
	ID       int64
	PublicID string
	Username string
}

func newProfileID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		buf[0:4],
		buf[4:6],
		buf[6:8],
		buf[8:10],
		buf[10:16],
	), nil
}

func (s *server) userFromRequest(r *http.Request) (userProfile, error) {
	profileID := strings.TrimSpace(r.Header.Get("X-Profile-ID"))
	if profileID == "" {
		cookie, err := r.Cookie("numbers_profile_id")
		if err == nil {
			decoded, decodeErr := url.QueryUnescape(strings.TrimSpace(cookie.Value))
			if decodeErr == nil {
				profileID = decoded
			}
		}
	}
	if profileID == "" {
		return userProfile{}, fmt.Errorf("missing profile id")
	}

	return s.ensureUserByProfileID(r.Context(), profileID)
}

func (s *server) ensureUserByProfileID(ctx context.Context, profileID string) (userProfile, error) {
	if len(profileID) < 8 {
		return userProfile{}, fmt.Errorf("invalid profile id")
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return userProfile{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	defaultUsername := "Player-" + strings.ToUpper(strings.ReplaceAll(profileID[:8], "-", ""))
	_, err = tx.Exec(ctx, `
		INSERT INTO users (public_id, username)
		VALUES ($1, $2)
		ON CONFLICT (public_id) DO NOTHING
	`, profileID, defaultUsername)
	if err != nil {
		return userProfile{}, fmt.Errorf("upsert profile user: %w", err)
	}

	var user userProfile
	err = tx.QueryRow(ctx, `
		SELECT id, public_id, username
		FROM users
		WHERE public_id = $1
	`, profileID).Scan(&user.ID, &user.PublicID, &user.Username)
	if err != nil {
		return userProfile{}, fmt.Errorf("select profile user: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO user_roll_state (user_id, rolls_available, last_refill_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (user_id) DO NOTHING
	`, user.ID, s.cfg.RollMaxTokens)
	if err != nil {
		return userProfile{}, fmt.Errorf("ensure roll state: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return userProfile{}, fmt.Errorf("commit tx: %w", err)
	}

	return user, nil
}

func (s *server) updateUsername(ctx context.Context, userID int64, username string) (userProfile, error) {
	username = strings.TrimSpace(username)

	var updated userProfile
	err := s.db.QueryRow(ctx, `
		UPDATE users
		SET username = $2
		WHERE id = $1
		RETURNING id, public_id, username
	`, userID, username).Scan(&updated.ID, &updated.PublicID, &updated.Username)
	if err != nil {
		return userProfile{}, fmt.Errorf("update username: %w", err)
	}

	return updated, nil
}
