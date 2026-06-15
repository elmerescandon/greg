package metrics

import (
	"time"

	"github.com/elmerescandon/greg-ui/internal/session"
)

type DayStat struct {
	Date     string
	Sessions int
	Cost     float64
}

type Summary struct {
	MonthSessions int
	MonthCost     float64
	WeekSessions  int
	WeekCost      float64
	TodaySessions int
	TodayCost     float64
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

func DailyStats(sessions []session.Session, days int) []DayStat {
	byDay := map[string]*DayStat{}
	now := time.Now()
	cutoff := now.AddDate(0, 0, -(days - 1))
	cutoff = time.Date(cutoff.Year(), cutoff.Month(), cutoff.Day(), 0, 0, 0, 0, cutoff.Location())

	for _, s := range sessions {
		t, err := time.Parse("2006-01-02 15:04:05", s.Started)
		if err != nil || t.Before(cutoff) {
			continue
		}
		key := t.Format("2006-01-02")
		if byDay[key] == nil {
			byDay[key] = &DayStat{Date: key}
		}
		byDay[key].Sessions++
		byDay[key].Cost += s.CostUSD
	}

	result := make([]DayStat, days)
	for i := 0; i < days; i++ {
		d := now.AddDate(0, 0, -(days - 1 - i))
		key := d.Format("2006-01-02")
		if stat, ok := byDay[key]; ok {
			result[i] = *stat
		} else {
			result[i] = DayStat{Date: key}
		}
	}
	return result
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
		}
		if !t.Before(weekStart) {
			s.WeekSessions++
			s.WeekCost += sess.CostUSD
		}
		if !t.Before(todayStart) {
			s.TodaySessions++
			s.TodayCost += sess.CostUSD
		}
	}
	return s
}
