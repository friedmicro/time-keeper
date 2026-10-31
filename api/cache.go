package main

import (
	"net/http"
	"sync"
)

type lastResponseCache struct {
	mu sync.Mutex
	m  map[string][]byte
}

type capture struct {
	http.ResponseWriter
	status    int
	body      []byte
	wroteHead bool
}

func newLastResponseCache() *lastResponseCache {
	return &lastResponseCache{m: make(map[string][]byte)}
}

func (c *lastResponseCache) set(key string, body []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[key] = body
}

func (c *lastResponseCache) get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	b, ok := c.m[key]
	return b, ok
}

func (c *capture) WriteHeader(code int) {
	c.status = code
	c.wroteHead = true
}

func (c *capture) Write(p []byte) (int, error) {
	c.body = append(c.body, p...)
	return len(p), nil
}

func (c *lastResponseCache) wrap(key string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rec := &capture{ResponseWriter: w, status: http.StatusOK}
		next(rec, r)
		if !rec.wroteHead {
			rec.status = http.StatusOK
		}

		if r.Method == http.MethodGet {
			if rec.status >= 200 && rec.status < 300 {
				c.set(key, rec.body)
			} else if rec.status >= 400 {
				if stale, ok := c.get(key); ok {
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("X-TimeKeeper-Stale", "true")
					w.WriteHeader(http.StatusOK)
					w.Write(stale)
					return
				}
			}
		}

		w.WriteHeader(rec.status)
		w.Write(rec.body)
	}
}
