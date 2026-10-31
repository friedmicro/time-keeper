package main

import (
	"log"
	"net/http"
	"os"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type Config struct {
	DBHost             string
	DBUser             string
	DBPassword         string
	DBName             string
	Port               string
	SerpentsClientIP   string
	SerpentsWebhookURL string
	Timezone           string
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func loadConfig() Config {
	return Config{
		DBHost:             env("DB_HOST", "localhost"),
		DBUser:             env("DB_USER", "root"),
		DBPassword:         env("DB_PASSWORD", ""),
		DBName:             env("DB_NAME", ""),
		Port:               env("PORT", "5006"),
		SerpentsClientIP:   env("SERPENTS_CLIENT_IP", ""),
		SerpentsWebhookURL: env("SERPENTS_WEBHOOK_URL", ""),
		Timezone:           env("TIMEZONE", "America/Los_Angeles"),
	}
}

func main() {
	cfg := loadConfig()

	if err := initDB(cfg); err != nil {
		log.Fatalf("init db: %v", err)
	}

	mux := http.NewServeMux()
	cache := newLastResponseCache()
	mux.HandleFunc("/api/week_times", cache.wrap("/api/week_times", handleWeekTimes(cfg)))
	mux.HandleFunc("/api/week", cache.wrap("/api/week", handleWeek(cfg)))
	mux.HandleFunc("/api/current", cache.wrap("/api/current", handleCurrent(cfg)))
	mux.HandleFunc("/api/day", handleDay(cfg))
	mux.HandleFunc("/api/start", handleStart(cfg))
	mux.HandleFunc("/api/stop", handleStop(cfg))
	mux.HandleFunc("/api/add_8_hours", handleAdd8Hours(cfg))
	mux.HandleFunc("/api/add_time", handleAddTime(cfg))

	addr := "0.0.0.0:" + cfg.Port
	log.Printf("TimeKeeper API listening on %s", addr)
	if err := http.ListenAndServe(addr, corsMiddleware(mux)); err != nil {
		log.Fatal(err)
	}
}
