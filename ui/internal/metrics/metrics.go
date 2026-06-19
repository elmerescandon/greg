package metrics

import (
	"fmt"
	"path/filepath"
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
	PrevCost           float64
	PrevSessions       int
}

type MonthSummary struct {
	Label     string
	Year      int
	Month     time.Month
	Sessions  int
	Cost      float64
	Tokens    int
	IsCurrent bool
}

type DaySummary struct {
	Date     string
	Cost     float64
	Sessions int
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
	prevMonthStart := monthStart.AddDate(0, -1, 0)
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
		} else if !t.Before(prevMonthStart) {
			s.PrevSessions++
			s.PrevCost += sess.CostUSD
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

func MonthlyHistory(sessions []session.Session) []MonthSummary {
	now := time.Now()
	type key struct{ year int; month time.Month }
	counts := map[key]*MonthSummary{}

	for _, sess := range sessions {
		t, err := time.Parse("2006-01-02 15:04:05", sess.Started)
		if err != nil {
			continue
		}
		k := key{t.Year(), t.Month()}
		if _, ok := counts[k]; !ok {
			counts[k] = &MonthSummary{
				Year:  t.Year(),
				Month: t.Month(),
				Label: fmt.Sprintf("%s %d", t.Month().String()[:3], t.Year()),
			}
		}
		counts[k].Sessions++
		counts[k].Cost += sess.CostUSD
		counts[k].Tokens += sess.InputTokens + sess.OutputTokens
	}

	currentKey := key{now.Year(), now.Month()}
	var result []MonthSummary
	for i := 5; i >= 0; i-- {
		d := now.AddDate(0, -i, 0)
		k := key{d.Year(), d.Month()}
		if ms, ok := counts[k]; ok {
			ms.IsCurrent = (k == currentKey)
			result = append(result, *ms)
		}
	}
	return result
}

func ByModel(sessions []session.Session) map[string]Summary {
	result := map[string]Summary{}
	for _, sess := range sessions {
		model := sess.Model
		if model == "" {
			model = "desconocido"
		}
		s := result[model]
		s.MonthSessions++
		s.MonthCost += sess.CostUSD
		s.MonthInputTokens += sess.InputTokens
		s.MonthOutputTokens += sess.OutputTokens
		result[model] = s
	}
	return result
}

func ByProject(sessions []session.Session) map[string]Summary {
	result := map[string]Summary{}
	for _, sess := range sessions {
		project := "desconocido"
		if sess.Dir != "" {
			project = filepath.Base(sess.Dir)
		}
		s := result[project]
		s.MonthSessions++
		s.MonthCost += sess.CostUSD
		s.MonthInputTokens += sess.InputTokens
		s.MonthOutputTokens += sess.OutputTokens
		result[project] = s
	}
	return result
}

func DailyCostDist(sessions []session.Session, days int) []DaySummary {
	now := time.Now()
	result := make([]DaySummary, days)
	for i := 0; i < days; i++ {
		d := now.AddDate(0, 0, -(days - 1 - i))
		result[i].Date = d.Format("2006-01-02")
	}
	dateIndex := map[string]int{}
	for i, ds := range result {
		dateIndex[ds.Date] = i
	}
	for _, sess := range sessions {
		t, err := time.Parse("2006-01-02 15:04:05", sess.Started)
		if err != nil {
			continue
		}
		dateStr := t.Format("2006-01-02")
		if idx, ok := dateIndex[dateStr]; ok {
			result[idx].Cost += sess.CostUSD
			result[idx].Sessions++
		}
	}
	return result
}
