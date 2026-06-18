package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
	_ "time/tzdata"
)

const (
	apiTimeout  = 5 * time.Second
	pushTimeout = 20 * time.Second
	pollEvery   = 5 * time.Second
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

type config struct {
	apiBase   string
	timezone  string
	pushcut   string
	states    map[string]string
	texts     map[string]string
	workHours float64
	lowHours  float64
}

func loadConfig() config {
	return config{
		apiBase:  env("TIMEKEEPER_API", ""),
		timezone: env("TIMEZONE", "America/Los_Angeles"),
		pushcut:  env("PUSHCUT_URL", ""),
		states: map[string]string{
			"work":     env("STATE_WORK", "lucia_work"),
			"lowHours": env("STATE_LOW_HOURS", "lucia_low_hours"),
			"notWork":  env("STATE_NOT_WORK", "lucia_not_work"),
			"free":     env("STATE_FREE", "lucia_free"),
		},
		texts: map[string]string{
			"work":     env("STATE_WORK_TEXT", "Lucia is working"),
			"lowHours": env("STATE_LOW_HOURS_TEXT", "Lucia is low on hours"),
			"notWork":  env("STATE_NOT_WORK_TEXT", "Lucia is not working"),
			"free":     env("STATE_FREE_TEXT", "Lucia is free for day"),
		},
		workHours: envFloat("WORK_HOURS", 8),
		lowHours:  envFloat("WORK_LOW_HOURS", 6),
	}
}

func decide(cfg config, running bool, totalHours float64, now time.Time) (state, text string) {
	if running {
		return cfg.states["work"], cfg.texts["work"]
	}
	dayCount := int(now.Weekday())
	dayCount = ((dayCount + 6) % 7) + 1
	if dayCount > 5 {
		dayCount = 5
	}
	dayTotal := float64(dayCount)
	switch {
	case dayTotal*cfg.lowHours > totalHours:
		return cfg.states["lowHours"], cfg.texts["lowHours"]
	case dayTotal*cfg.workHours > totalHours:
		return cfg.states["notWork"], cfg.texts["notWork"]
	default:
		return cfg.states["free"], cfg.texts["free"]
	}
}

type currentResponse struct {
	Running bool `json:"running"`
}

type weekResponse struct {
	Totals float64 `json:"totals"`
}

func fetch(client *http.Client, cfg config, now time.Time) (state, text string, err error) {
	cur, err := client.Get(cfg.apiBase + "/api/current")
	if err != nil {
		return "", "", err
	}
	defer cur.Body.Close()

	wk, err := client.Get(cfg.apiBase + "/api/week_times")
	if err != nil {
		return "", "", err
	}
	defer wk.Body.Close()

	if cur.StatusCode != http.StatusOK || wk.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("api returned current=%s week=%s", cur.Status, wk.Status)
	}

	var running currentResponse
	if err := json.NewDecoder(cur.Body).Decode(&running); err != nil {
		return "", "", fmt.Errorf("decode current: %w", err)
	}
	var total weekResponse
	if err := json.NewDecoder(wk.Body).Decode(&total); err != nil {
		return "", "", fmt.Errorf("decode week: %w", err)
	}
	state, text = decide(cfg, running.Running, total.Totals, now)
	return state, text, nil
}

func pushcut(client *http.Client, cfg config, state, text string) string {
	body, err := json.Marshal(map[string]string{"input": state, "text": text})
	if err != nil {
		log.Printf("marshal pushcut payload: %v", err)
		return ""
	}
	resp, err := client.Post(cfg.pushcut, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("pushcut error: %v", err)
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("pushcut rejected: %s", resp.Status)
		return ""
	}
	var reply any
	_ = json.NewDecoder(resp.Body).Decode(&reply)
	log.Printf("pushcut ok: %v", reply)
	return state
}

func main() {
	cfg := loadConfig()
	loc, err := time.LoadLocation(cfg.timezone)
	if err != nil {
		log.Fatalf("load location %q: %v", cfg.timezone, err)
	}

	client := &http.Client{Timeout: apiTimeout}
	pushClient := &http.Client{Timeout: pushTimeout}

	var lastPushed string
	for {
		now := time.Now().In(loc)
		state, text, err := fetch(client, cfg, now)
		if err != nil {
			log.Printf("fetch error: %v", err)
			time.Sleep(pollEvery)
			continue
		}
		if state != lastPushed {
			lastPushed = pushcut(pushClient, cfg, state, text)
		}
		time.Sleep(pollEvery)
	}
}
