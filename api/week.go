package main

import (
	"database/sql"
	"time"
)

type TimeRow struct {
	Date         time.Time
	TotalSeconds float64
	Category     string
}

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

func getTimeData(db *sql.DB, cfg Config, now time.Time) ([]TimeRow, error) {
	monday := getMonday(now)
	rows, err := db.Query(`
		SELECT DATE(start_time) AS date, SUM(TIMESTAMPDIFF(SECOND, start_time, COALESCE(end_time, NOW()))) AS total_seconds
		FROM sessions
		WHERE end_time IS NOT NULL
		  AND DATE(start_time) >= ?
		GROUP BY DATE(start_time)
		ORDER BY date DESC`, monday)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []TimeRow
	for rows.Next() {
		var r TimeRow
		var d time.Time
		if err := rows.Scan(&d, &r.TotalSeconds); err != nil {
			return nil, err
		}
		r.Date = d
		data = append(data, r)
	}
	return data, rows.Err()
}

func getPreviousWeekExtraHours(db *sql.DB, cfg Config, now time.Time) (int64, error) {
	monday := getMonday(now)
	prevMonday := monday.AddDate(0, 0, -7)

	var total sql.NullFloat64
	err := db.QueryRow(`
		SELECT SUM(TIMESTAMPDIFF(SECOND, start_time, COALESCE(end_time, NOW())))
		FROM sessions
		WHERE end_time IS NOT NULL
		  AND DATE(start_time) >= ?
		  AND DATE(start_time) < ?`, prevMonday, monday).Scan(&total)
	if err != nil {
		return 0, err
	}

	var t int64
	if total.Valid {
		t = int64(total.Float64)
	}

	const weekHours = 40 * 60 * 60
	if t > weekHours {
		return t - weekHours, nil
	}
	return 0, nil
}

func getWeekTotals(db *sql.DB, cfg Config, now time.Time) (int64, error) {
	monday := getMonday(now)
	var total sql.NullFloat64
	err := db.QueryRow(`
		SELECT SUM(TIMESTAMPDIFF(SECOND, start_time, COALESCE(end_time, NOW())))
		FROM sessions
		WHERE end_time IS NOT NULL
		  AND DATE(start_time) >= ?`, monday).Scan(&total)
	if err != nil {
		return 0, err
	}
	var t int64
	if total.Valid {
		t = int64(total.Float64)
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

func getCategoryRows(db *sql.DB, cfg Config, now time.Time) ([]TimeRow, error) {
	monday := getMonday(now)
	rows, err := db.Query(`
		SELECT DATE(start_time) AS date, category, SUM(TIMESTAMPDIFF(SECOND, start_time, COALESCE(end_time, NOW()))) AS total_seconds
		FROM sessions
		WHERE end_time IS NOT NULL
		  AND DATE(start_time) >= ?
		GROUP BY DATE(start_time), category`, monday)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []TimeRow
	for rows.Next() {
		var r TimeRow
		var d time.Time
		if err := rows.Scan(&d, &r.Category, &r.TotalSeconds); err != nil {
			return nil, err
		}
		r.Date = d
		data = append(data, r)
	}
	return data, rows.Err()
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
