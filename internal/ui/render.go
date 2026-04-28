package ui

import (
	"fmt"
	"html"
	"strings"
	"time"
)

type ProfileEntry struct {
	PublicID string
	Username string
}

type HistoryEntry struct {
	Number    int
	Score     int
	CreatedAt time.Time
}

type UnlockedSpecEntry struct {
	SpecKey   string
	RollCount int
}

type LeaderboardEntry struct {
	Username   string
	BestScore  int
	RollCount  int
	BestNumber int
}

type TotalValueLeaderboardEntry struct {
	Username   string
	TotalValue int
	RollCount  int
	BestNumber int
}

type RollStatus struct {
	Available         int
	NextRollInSeconds int
}

type Spec struct {
	Key   string
	Name  string
	Value string
	Score int
}

var rareSpecs = map[string]struct{}{
	"prime":              {},
	"all_digits_equal":   {},
	"trailing_000":       {},
	"palindrome":         {},
	"three_consecutive":  {},
}

func RenderProfilePanel(user ProfileEntry) string {
	return RenderProfilePanelWithMessage(user, "")
}

func RenderProfilePanelWithMessage(user ProfileEntry, message string) string {
	var b strings.Builder
	b.WriteString("<div class=\"panel\">")
	b.WriteString("<h2>Profile</h2>")
	b.WriteString(fmt.Sprintf("<p class=\"meta\">Profile ID: <code>%s</code></p>", html.EscapeString(user.PublicID)))
	if strings.TrimSpace(message) != "" {
		b.WriteString(fmt.Sprintf("<p class=\"meta\"><strong>%s</strong></p>", html.EscapeString(message)))
	}
	b.WriteString("<form hx-post=\"/profile/username\" hx-target=\"#profile-panel\" hx-swap=\"innerHTML\">")
	b.WriteString("<label for=\"username\">Username</label>")
	b.WriteString(fmt.Sprintf("<input id=\"username\" name=\"username\" maxlength=\"24\" value=\"%s\" required />", html.EscapeString(user.Username)))
	b.WriteString("<button type=\"submit\">Save Username</button>")
	b.WriteString("</form>")
	b.WriteString("</div>")
	return b.String()
}

func RenderPanelMessage(title, message string) string {
	var b strings.Builder
	b.WriteString("<div class=\"panel\">")
	b.WriteString(fmt.Sprintf("<h2>%s</h2>", html.EscapeString(title)))
	b.WriteString(fmt.Sprintf("<p class=\"meta\">%s</p>", html.EscapeString(message)))
	b.WriteString("</div>")
	return b.String()
}

func RenderHistoryPanel(items []HistoryEntry) string {
	var b strings.Builder
	b.WriteString("<div class=\"panel\">")
	b.WriteString("<h2>Roll History</h2>")
	if len(items) == 0 {
		b.WriteString("<p class=\"meta\">No rolls yet.</p>")
		b.WriteString("</div>")
		return b.String()
	}

	b.WriteString("<ul class=\"simple-list\">")
	for _, item := range items {
		b.WriteString("<li>")
		b.WriteString(fmt.Sprintf("<span>%06d</span>", item.Number))
		b.WriteString(fmt.Sprintf("<strong>%d pts</strong>", item.Score))
		b.WriteString(fmt.Sprintf("<small>%s</small>", item.CreatedAt.Local().Format("2006-01-02 15:04:05")))
		b.WriteString("</li>")
	}
	b.WriteString("</ul>")
	b.WriteString("</div>")
	return b.String()
}

func RenderUnlockedSpecsPanel(items []UnlockedSpecEntry) string {
	var b strings.Builder
	b.WriteString("<div class=\"panel\">")
	b.WriteString("<h2>Unlocked Specs</h2>")
	if len(items) == 0 {
		b.WriteString("<p class=\"meta\">No unlocked specs yet.</p>")
		b.WriteString("</div>")
		return b.String()
	}

	b.WriteString("<ul class=\"simple-list\">")
	for _, item := range items {
		b.WriteString("<li>")
		displayName := getSpecDisplayName(item.SpecKey)
		if isRareSpec(item.SpecKey) {
			b.WriteString(fmt.Sprintf("<span class=\"spec-rare\">%s</span>", html.EscapeString(displayName)))
		} else {
			b.WriteString(fmt.Sprintf("<span>%s</span>", html.EscapeString(displayName)))
		}
		b.WriteString(fmt.Sprintf("<strong>x%d</strong>", item.RollCount))
		b.WriteString("<small>rolls</small>")
		b.WriteString("</li>")
	}
	b.WriteString("</ul>")
	b.WriteString("</div>")
	return b.String()
}

func RenderLeaderboardPanel(items []LeaderboardEntry) string {
	var b strings.Builder
	b.WriteString("<div class=\"panel\">")
	b.WriteString("<h2>Max value</h2>")
	if len(items) == 0 {
		b.WriteString("<p class=\"meta\">No leaderboard entries yet.</p>")
		b.WriteString("</div>")
		return b.String()
	}

	b.WriteString("<ol class=\"leader-list\">")
	for _, item := range items {
		b.WriteString("<li>")
		b.WriteString(fmt.Sprintf("<span>%s</span>", html.EscapeString(item.Username)))
		b.WriteString(fmt.Sprintf("<strong>%d pts |</strong>", item.BestScore))
		b.WriteString(fmt.Sprintf("<strong>%06d</strong>", item.BestNumber))
		b.WriteString(fmt.Sprintf("<small>%d rolls</small>", item.RollCount))
		b.WriteString("</li>")
	}
	b.WriteString("</ol>")
	b.WriteString("</div>")
	return b.String()
}

func RenderTotalValueLeaderboardPanel(items []TotalValueLeaderboardEntry) string {
	var b strings.Builder
	b.WriteString("<div class=\"panel\">")
	b.WriteString("<h2>Total Value</h2>")
	if len(items) == 0 {
		b.WriteString("<p class=\"meta\">No entries yet.</p>")
		b.WriteString("</div>")
		return b.String()
	}

	b.WriteString("<ol class=\"leader-list\">")
	for _, item := range items {
		b.WriteString("<li>")
		b.WriteString(fmt.Sprintf("<span>%s</span>", html.EscapeString(item.Username)))
		b.WriteString(fmt.Sprintf("<strong>%d pts</strong>", item.TotalValue))
		b.WriteString(fmt.Sprintf("<small>%d rolls</small>", item.RollCount))
		b.WriteString("</li>")
	}
	b.WriteString("</ol>")
	b.WriteString("</div>")
	return b.String()
}

func RenderRollControls(status RollStatus, maxTokens int) string {
	var b strings.Builder

	b.WriteString("<div class=\"roll-controls\">")
	b.WriteString(fmt.Sprintf("<p class=\"meta\">Rolls: <strong>%d</strong> / %d</p>", status.Available, maxTokens))
	b.WriteString("<p class=\"meta\">Next +1 roll in: <strong id=\"next-roll-timer\" data-next-roll-seconds=\"")
	b.WriteString(fmt.Sprintf("%d\">", status.NextRollInSeconds))
	b.WriteString(formatDurationMMSS(status.NextRollInSeconds))
	b.WriteString("</strong></p>")

	if status.Available > 0 {
		b.WriteString("<button hx-post=\"/roll\" hx-target=\"#roll-result\" hx-swap=\"innerHTML\">Roll Number</button>")
	} else {
		b.WriteString("<button disabled>Out of Rolls</button>")
	}

	b.WriteString("</div>")
	return b.String()
}

func RenderNoRollsFragment(seconds int) string {
	return fmt.Sprintf(
		"<div class=\"roll-card\"><h3>No rolls left</h3><p>Your next roll arrives in <strong>%s</strong>.</p></div>",
		formatDurationMMSS(seconds),
	)
}

func RenderNeedsProfileFragment() string {
	return "<div class=\"roll-card\"><h3>Profile Not Ready</h3><p>Initializing profile, please wait a moment.</p></div>"
}

func RenderRollFragment(number int, specs []Spec, total int) string {
	var b strings.Builder
	b.WriteString("<div class=\"roll-card\">")
	b.WriteString("<h3>Rolled Number</h3>")
	b.WriteString(fmt.Sprintf("<p class=\"value\">%06d</p>", number))
	if len(specs) == 0 {
		b.WriteString("<p>No bonus specs hit.</p>")
	} else {
		b.WriteString("<ul>")
		for _, item := range specs {
			b.WriteString("<li>")
			if isRareSpec(item.Key) {
				b.WriteString(fmt.Sprintf("<span class=\"spec-rare\">%s: %s</span>", item.Name, item.Value))
			} else {
				b.WriteString(fmt.Sprintf("<span>%s: %s</span>", item.Name, item.Value))
			}
			b.WriteString(fmt.Sprintf("<strong>+%d</strong>", item.Score))
			b.WriteString("</li>")
		}
		b.WriteString("</ul>")
	}
	b.WriteString(fmt.Sprintf("<p class=\"total\">Total Score: %d</p>", total))
	b.WriteString("</div>")
	return b.String()
}

func formatDurationMMSS(totalSeconds int) string {
	if totalSeconds < 0 {
		totalSeconds = 0
	}
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func getSpecDisplayName(key string) string {
	specNames := map[string]string{
		"prime":              "Prime Time",
		"all_digits_equal":   "Six of a Kind",
		"palindrome":         "Mirror Number",
		"unique_digits_6":    "Sextuplet",
		"unique_digits_5":    "Quintuplet",
		"unique_digits_4":    "Quadruplet",
		"divisible_by_9":     "Nine Lives",
		"divisible_by_7":     "Lucky Sevens",
		"trailing_000":       "Trailing Zeroes",
		"exact_zero":         "The Big Zero",
		"even_number":        "Even Steven",
		"odd_number":         "Odd Todd",
		"starts_with_1":      "Leading One",
		"ends_with_5":        "High Five",
		"contains_7":         "Lucky Seven",
		"digit_sum_under_15": "Digit Sum Lite",
		"digit_sum_over_30":  "Digit Sum Heavy",
		"double_digit":       "Twin Digits",
		"fibonacci":          "Fibber",
		"perfect_square":     "Square Root",
		"armstrong":          "Armstrong's Strong",
		"ascending_digits":   "Climbing Digits",
		"descending_digits":  "Descending Digits",
		"contains_123":       "Counting Up",
		"contains_456":       "Mid-Count",
		"digit_product_24":   "Dozen Doubled",
		"alternating_parity": "Odd-Even Shift",
		"three_consecutive":  "Three in a Row",
	}
	if name, ok := specNames[key]; ok {
		return name
	}
	return humanizeSpecKey(key)
}

func humanizeSpecKey(key string) string {
	parts := strings.Split(key, "_")
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, " ")
}

func isRareSpec(key string) bool {
	_, ok := rareSpecs[key]
	return ok
}
