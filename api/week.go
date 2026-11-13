package main

import (
	"database/sql"
	"sort"
	"time"
)

var LA = mustLoadLocation(env("TIMEZONE", "America/Los_Angeles"))

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

func laNow() time.Time {
	return time.Now().In(LA)
}

func daysSinceMonday(t time.Time) int {
	return (int(t.Weekday()) + 6) % 7
}

func getMonday(now time.Time) time.Time {
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, LA)
	return start.AddDate(0, 0, -daysSinceMonday(start))
}

func sessionSeconds(s *SessionRow, now time.Time) int64 {
	start := s.StartTime
	return int64(now.In(LA).Sub(start).Seconds())
}

func toLAInstant(stored time.Time) time.Time {
	return time.Date(
		stored.Year(), stored.Month(), stored.Day(),
		stored.Hour(), stored.Minute(), stored.Second(),
		0, LA,
	)
}

func startOfLADay(t time.Time) time.Time {
	lt := t.In(LA)
	return time.Date(lt.Year(), lt.Month(), lt.Day(), 0, 0, 0, 0, LA)
}

func loadSessions(db *sql.DB, from time.Time) ([]SessionRecord, error) {
	rows, err := db.Query(`
		SELECT start_time, end_time, category
		FROM sessions
		WHERE end_time IS NOT NULL
		  AND start_time >= ?`,
		from.In(LA).Format("2006-01-02 15:04:05"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recs []SessionRecord
	for rows.Next() {
		var s, e time.Time
		var cat string
		if err := rows.Scan(&s, &e, &cat); err != nil {
			return nil, err
		}
		recs = append(recs, SessionRecord{
			Start:    toLAInstant(s),
			End:      toLAInstant(e),
			Category: cat,
		})
	}
	return recs, rows.Err()
}

func bucketSessions(sessions []SessionRecord) (map[time.Time]float64, map[time.Time]map[string]float64) {
	daysWithoutCategory := map[time.Time]float64{}
	daysWithCategory := map[time.Time]map[string]float64{}
	for _, s := range sessions {
		if !s.End.After(s.Start) {
			continue
		}
		day := startOfLADay(s.Start)
		for !day.After(s.End) {
			next := day.AddDate(0, 0, 1)
			overlapStart := s.Start
			if day.After(overlapStart) {
				overlapStart = day
			}
			overlapEnd := s.End
			if next.Before(overlapEnd) {
				overlapEnd = next
			}
			if overlapEnd.After(overlapStart) {
				daysWithoutCategory[day] += overlapEnd.Sub(overlapStart).Seconds()
				if daysWithCategory[day] == nil {
					daysWithCategory[day] = map[string]float64{}
				}
				daysWithCategory[day][s.Category] += overlapEnd.Sub(overlapStart).Seconds()
			}
			day = next
		}
	}
	return daysWithoutCategory, daysWithCategory
}

func getTimeData(db *sql.DB, now time.Time) ([]TimeRow, error) {
	monday := getMonday(now)
	sessions, err := loadSessions(db, monday.AddDate(0, 0, -1))
	if err != nil {
		return nil, err
	}
	var data []TimeRow
	buckets, _ := bucketSessions(sessions)
	for day, secs := range buckets {
		if !day.Before(monday) {
			data = append(data, TimeRow{Date: day, TotalSeconds: secs})
		}
	}
	sort.Slice(data, func(i, j int) bool { return data[i].Date.After(data[j].Date) })
	return data, nil
}

func getPreviousWeekExtraHours(db *sql.DB, cfg Config, now time.Time) (int64, error) {
	monday := getMonday(now)
	prevMonday := monday.AddDate(0, 0, -7)
	sessions, err := loadSessions(db, prevMonday.AddDate(0, 0, -1))
	if err != nil {
		return 0, err
	}

	var t int64
	buckets, _ := bucketSessions(sessions)
	for day, secs := range buckets {
		if !day.Before(prevMonday) && day.Before(monday) {
			t += int64(secs)
		}
	}

	const weekHours = 40 * 60 * 60
	if t > weekHours {
		return t - weekHours, nil
	}
	return 0, nil
}

func getWeekTotals(db *sql.DB, cfg Config, now time.Time) (int64, error) {
	monday := getMonday(now)
	sessions, err := loadSessions(db, monday.AddDate(0, 0, -1))
	if err != nil {
		return 0, err
	}

	var t int64
	buckets, _ := bucketSessions(sessions)
	for day, secs := range buckets {
		if !day.Before(monday) {
			t += int64(secs)
		}
	}

	previousWeekExtra, err := getPreviousWeekExtraHours(db, cfg, now)
	if err != nil {
		return 0, err
	}
	t += previousWeekExtra

	sess, err := getSession(db)
	if err != nil {
		return t, err
	}
	if sess != nil {
		t += sessionSeconds(sess, now)
	}
	return t, nil
}

func getCategoryRows(db *sql.DB, now time.Time) ([]TimeRow, error) {
	monday := getMonday(now)
	sessions, err := loadSessions(db, monday.AddDate(0, 0, -1))
	if err != nil {
		return nil, err
	}
	var data []TimeRow
	_, buckets := bucketSessions(sessions)
	for day, cats := range buckets {
		if day.Before(monday) {
			continue
		}
		for cat, secs := range cats {
			data = append(data, TimeRow{Date: day, TotalSeconds: secs, Category: cat})
		}
	}
	sort.Slice(data, func(i, j int) bool {
		if data[i].Date.Equal(data[j].Date) {
			return data[i].Category < data[j].Category
		}
		return data[i].Date.After(data[j].Date)
	})
	return data, nil
}

func processPendingTimes(data []TimeRow, sess *SessionRow, now time.Time) []TimeRow {
	if sess == nil {
		return data
	}
	cur := sessionSeconds(sess, now)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, LA)
	added := false
	for i := range data {
		if sameDay(data[i].Date, today) {
			data[i].TotalSeconds += float64(cur)
			added = true
			break
		}
	}
	if !added {
		data = append([]TimeRow{{Date: today, TotalSeconds: float64(cur)}}, data...)
	}
	return data
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.In(LA).Date()
	by, bm, bd := b.In(LA).Date()
	return ay == by && am == bm && ad == bd
}
