package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
	_ "time/tzdata"
)

const httpTimeout = 5 * time.Second

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

var (
	addr          = env("TIME_STATE_LISTEN", ":5000")
	timeBase      = env("TIMEKEEPER_API", "http://time.lucia.friedmicro-lab.org")
	stateWork     = env("STATE_WORK", "lucia_work")
	stateLowHours = env("STATE_LOW_HOURS", "lucia_low_hours")
	stateNotWork  = env("STATE_NOT_WORK", "lucia_not_work")
	stateFree     = env("STATE_FREE", "lucia_free")
	textWork      = env("STATE_WORK_TEXT", "Lucia is working")
	textLowHours  = env("STATE_LOW_HOURS_TEXT", "Lucia is low on hours")
	textNotWork   = env("STATE_NOT_WORK_TEXT", "Lucia is not working")
	textFree      = env("STATE_FREE_TEXT", "Lucia is free for day")
	workHours     = env("WORK_HOURS", "8")
	workLowHours  = env("WORK_LOW_HOURS", "6")
	timezone      = env("TIMEZONE", "America/Los_Angeles")
)

var la = func() *time.Location {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		panic(err)
	}
	return loc
}()

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/state", handleState)
	fmt.Fprintf(os.Stderr, "lucia-time-state listening on %s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func getState() (string, string) {
	client := &http.Client{Timeout: httpTimeout}

	currentResp, err := client.Get(timeBase + "/api/current")
	if err != nil {
		return "", err.Error()
	}
	defer currentResp.Body.Close()

	hoursResp, err := client.Get(timeBase + "/api/week_times")
	if err != nil {
		return "", err.Error()
	}
	defer hoursResp.Body.Close()

	if currentResp.StatusCode != http.StatusOK {
		return "", fmt.Sprintf("Time API error: %s", http.StatusText(currentResp.StatusCode))
	}
	if hoursResp.StatusCode != http.StatusOK {
		return "", "Hours API error"
	}

	var current struct {
		Running bool `json:"running"`
	}
	if err := json.NewDecoder(currentResp.Body).Decode(&current); err != nil {
		return "", err.Error()
	}

	var week struct {
		Totals float64 `json:"totals"`
	}
	if err := json.NewDecoder(hoursResp.Body).Decode(&week); err != nil {
		return "", err.Error()
	}

	dayCount := int(time.Now().In(la).Weekday()) + 1
	if dayCount > 5 {
		dayCount = 5
	}

	lowHours, workHoursF := 6.0, 8.0
	if v, err := strconv.ParseFloat(workLowHours, 64); err == nil {
		lowHours = v
	}
	if v, err := strconv.ParseFloat(workHours, 64); err == nil {
		workHoursF = v
	}

	if current.Running {
		return stateWork, textWork
	}
	if float64(dayCount)*lowHours > week.Totals {
		return stateLowHours, textLowHours
	}
	if float64(dayCount)*workHoursF > week.Totals {
		return stateNotWork, textNotWork
	}
	return stateFree, textFree
}

func handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	state, msg := getState()
	if state == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": msg})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"state": state, "message": msg})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
