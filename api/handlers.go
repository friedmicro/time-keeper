package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"strings"
	"time"
)

func sprintf(format string, a ...any) string { return fmt.Sprintf(format, a...) }

func methodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func clientIP(r *http.Request) string {
	if xRealIP := r.Header.Get("X-Real-IP"); xRealIP != "" {
		return xRealIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func handleClientLogic(cfg Config, r *http.Request) {
	if clientIP(r) != cfg.SerpentsClientIP {
		return
	}
	log.Printf("serpents client detected (%s); firing webhook", cfg.SerpentsClientIP)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(cfg.SerpentsWebhookURL, "application/json", strings.NewReader("{}"))
	if err != nil {
		log.Printf("serpents webhook failed: %v", err)
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

func round2(x float64) float64 {
	return math.Round(x*100) / 100
}

func hms(total int64) string {
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	return sprintf("%dh %dm %ds", h, m, s)
}

func computeWeekFlags(weekHours float64) (bool, bool) {
	now := laNow()
	iso := isoWeekday(now)
	quotaHours := iso * 8
	if quotaHours > 40 {
		quotaHours = 40
	}
	breakHours := iso * 6
	if breakHours > 30 {
		breakHours = 30
	}
	breakMet := float64(breakHours) <= weekHours
	quotaMet := float64(quotaHours) <= weekHours
	if now.Hour() >= 0 && now.Hour() < 4 {
		breakMet = true
	}
	return breakMet, quotaMet
}

func computeDays(weekTimes []TimeRow) (float64, map[int]map[string]any) {
	now := laNow()
	days := map[int]map[string]any{}
	todayKey := daysSinceMonday(now)
	var todayTime float64
	for _, wt := range weekTimes {
		dateKey := daysSinceMonday(wt.Date)
		hours := round2(float64(int64(wt.TotalSeconds)) / 3600)
		days[dateKey] = map[string]any{
			"date":  wt.Date.In(LA).Format("2006-01-02"),
			"hours": hours,
		}
		if dateKey == todayKey {
			todayTime = hours
		}
	}
	return todayTime, days
}

func handleWeekTimes(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, "GET")
			return
		}
		now := laNow()
		db, err := getDB(cfg)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer db.Close()

		weekTimes, err := getTimeData(db, now)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		sess, err := getSession(db)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		weekTimes = processPendingTimes(weekTimes, sess, now)

		weekRow, err := getWeekTotals(db, cfg, now)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		weekHours := round2(float64(weekRow) / 3600)

		todayTime, days := computeDays(weekTimes)
		breakMet, quotaMet := computeWeekFlags(weekHours)

		writeJSON(w, http.StatusOK, map[string]any{
			"days":        days,
			"totals":      weekHours,
			"quota_met":   quotaMet,
			"break_met":   breakMet,
			"today_hours": todayTime,
		})
	}
}

func formSessionJSON(days []TimeRow, categories []TimeRow) ([]DayJSON, []CategoryJSON) {
	dayOut := make([]DayJSON, 0, len(days))
	for _, d := range days {
		dayOut = append(dayOut, DayJSON{
			Date:  d.Date.In(LA).Format("2006-01-02"),
			Total: hms(int64(d.TotalSeconds)),
		})
	}
	catOut := make([]CategoryJSON, 0, len(categories))
	for _, c := range categories {
		catOut = append(catOut, CategoryJSON{
			Date:     c.Date.In(LA).Format("2006-01-02"),
			Category: c.Category,
			Total:    hms(int64(c.TotalSeconds)),
		})
	}
	return dayOut, catOut
}

func handleWeek(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, "GET")
			return
		}
		now := laNow()
		db, err := getDB(cfg)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer db.Close()

		days, err := getTimeData(db, now)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		sess, err := getSession(db)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		days = processPendingTimes(days, sess, now)

		categories, err := getCategoryRows(db, now)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		categories = processPendingTimes(categories, sess, now)

		weekTotal, err := getWeekTotals(db, cfg, now)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		dayOut, catOut := formSessionJSON(days, categories)
		writeJSON(w, http.StatusOK, map[string]any{
			"week_total": hms(weekTotal),
			"days":       dayOut,
			"categories": catOut,
		})
	}
}

func handleCurrent(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, "GET")
			return
		}
		db, err := getDB(cfg)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer db.Close()
		sess, err := getSession(db)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if sess != nil {
			// Return UTC with Z suffix so clients parse as UTC (not local time).
			writeJSON(w, http.StatusOK, map[string]any{
				"running":    true,
				"started_at": sess.StartTime.UTC().Format("2006-01-02T15:04:05") + "Z",
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"running": false})
	}
}

func handleDay(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w, "POST")
			return
		}
		db, err := getDB(cfg)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer db.Close()
		existing, err := getSession(db)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if existing != nil {
			stopSession(db, existing.ID)
		} else {
			startSession(db, "uncategorized")
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func getCategoryParameter(r *http.Request) string {
	category := "uncategorized"
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
		if c, ok := body["category"].(string); ok && c != "" {
			category = c
		}
	}
	return category
}

func handleStart(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w, "POST")
			return
		}
		handleClientLogic(cfg, r)
		db, err := getDB(cfg)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer db.Close()
		existing, err := getSession(db)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if existing != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "message": "A session is already running"})
			return
		}
		category := getCategoryParameter(r)
		if err := startSession(db, category); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "message": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func handleStop(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w, "POST")
			return
		}
		handleClientLogic(cfg, r)
		db, err := getDB(cfg)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer db.Close()
		sess, err := getSession(db)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if sess == nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "message": "No running session"})
			return
		}
		if err := stopSession(db, sess.ID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "message": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func handleAdd8Hours(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w, "POST")
			return
		}
		now := laNow()
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, LA)
		end := start.Add(8 * time.Hour)
		id, err := insertSession(cfg, start, end, "life_event")
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "message": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"id":     id,
			"start":  start.Format("2006-01-02T15:04:05"),
			"end":    end.Format("2006-01-02T15:04:05"),
		})
	}
}

func getHandleAddTimeParams(r *http.Request) (float64, string, string) {
	var hours float64
	var category string
	var dateStr string
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
		category = getCategoryParameter(r)
		if h, ok := body["hours"].(float64); ok {
			hours = h
		}
		if d, ok := body["date"].(string); ok {
			dateStr = d
		}
	}
	return hours, dateStr, category
}

func handleAddTime(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w, "POST")
			return
		}
		hours, dateStr, category := getHandleAddTimeParams(r)
		day, err := addTimeDay(dateStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "message": err.Error()})
			return
		}
		start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, LA)
		end := start.Add(time.Duration(hours * float64(time.Hour)))
		id, err := insertSession(cfg, start, end, category)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "message": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"id":     id,
			"start":  start.Format("2006-01-02T15:04:05"),
			"end":    end.Format("2006-01-02T15:04:05"),
		})
	}
}

func startSession(db *sql.DB, category string) error {
	_, err := db.Exec("INSERT INTO sessions (start_time, category) VALUES (NOW(), ?)", category)
	return err
}

func addTimeDay(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return laNow(), nil
	}
	parsed, err := time.ParseInLocation("02-01-2006", dateStr, LA)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q, expected DD-MM-YYYY", dateStr)
	}
	return parsed, nil
}

func stopSession(db *sql.DB, id int64) error {
	_, err := db.Exec("UPDATE sessions SET end_time = NOW() WHERE id = ?", id)
	return err
}

func insertSession(cfg Config, start, end time.Time, category string) (int64, error) {
	db, err := getDB(cfg)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	res, err := db.Exec(
		"INSERT INTO sessions (start_time, end_time, category) VALUES (?, ?, ?)",
		start.In(LA).Format("2006-01-02 15:04:05"),
		end.In(LA).Format("2006-01-02 15:04:05"),
		category,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func isoWeekday(t time.Time) int {
	wd := int(t.Weekday())
	if wd == 0 {
		return 7
	}
	return wd
}
