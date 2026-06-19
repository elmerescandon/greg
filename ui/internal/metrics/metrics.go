package metrics

import (
	"time"

	"github.com/elmerescandon/greg-ui/internal/session"
)

type Summary struct {
	MonthSessions      int
	MonthCost          float64
	MonthInputTokens   int
	MonthOutputTokens  int
	WeekSessions       int
	WeekCost           float64
	WeekInputTokens    int
	WeekOutputTokens   int
	TodaySessions      int
	TodayCost          float64
	TodayInputTokens   int
	TodayOutputTokens  int
}

func AllSessions() []session.Session {
	active, _ := session.LoadSessions()
	finished, _ := session.LoadFinishedSessions()
	seen := map[string]bool{}
	var all []session.Session
	for _, s := range finished {
		if !seen[s.ID] {
			seen[s.ID] = true
			all = append(all, s)
		}
	}
	for _, s := range active {
		if !seen[s.ID] {
			seen[s.ID] = true
			all = append(all, s)
		}
	}
	return all
}

func HourlyDist(sessions []session.Session) [24]int {
	var dist [24]int
	for _, s := range sessions {
		t, err := time.Parse("2006-01-02 15:04:05", s.Started)
		if err != nil {
			continue
		}
		dist[t.Hour()]++
	}
	return dist
}

// HourlyCostDist returns total cost per hour.
func HourlyCostDist(sessions []session.Session) [24]float64 {
	var dist [24]float64
	for _, s := range sessions {
		t, err := time.Parse("2006-01-02 15:04:05", s.Started)
		if err != nil {
			continue
		}
		dist[t.Hour()] += s.CostUSD
	}
	return dist
}

// HourlyInputTokensDist returns total input tokens per hour.
func HourlyInputTokensDist(sessions []session.Session) [24]float64 {
	var dist [24]float64
	for _, s := range sessions {
		t, err := time.Parse("2006-01-02 15:04:05", s.Started)
		if err != nil {
			continue
		}
		dist[t.Hour()] += float64(s.InputTokens)
	}
	return dist
}

// HourlyOutputTokensDist returns total output tokens per hour.
func HourlyOutputTokensDist(sessions []session.Session) [24]float64 {
	var dist [24]float64
	for _, s := range sessions {
		t, err := time.Parse("2006-01-02 15:04:05", s.Started)
		if err != nil {
			continue
		}
		dist[t.Hour()] += float64(s.OutputTokens)
	}
	return dist
}

func GetSummary(sessions []session.Session) Summary {
	var s Summary
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	// week starts on Monday
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekStart := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	for _, sess := range sessions {
		t, err := time.Parse("2006-01-02 15:04:05", sess.Started)
		if err != nil {
			continue
		}
		if !t.Before(monthStart) {
			s.MonthSessions++
			s.MonthCost += sess.CostUSD
			s.MonthInputTokens += sess.InputTokens
			s.MonthOutputTokens += sess.OutputTokens
		}
		if !t.Before(weekStart) {
			s.WeekSessions++
			s.WeekCost += sess.CostUSD
			s.WeekInputTokens += sess.InputTokens
			s.WeekOutputTokens += sess.OutputTokens
		}
		if !t.Before(todayStart) {
			s.TodaySessions++
			s.TodayCost += sess.CostUSD
			s.TodayInputTokens += sess.InputTokens
			s.TodayOutputTokens += sess.OutputTokens
		}
	}
	return s
}
