package main

import (
	"database/sql"
	"time"
)

type SessionRow struct {
	ID        int64
	StartTime time.Time
	EndTime   sql.NullTime
	Category  string
}

type TimeRow struct {
	Date         time.Time
	TotalSeconds float64
	Category     string
}

type SessionRecord struct {
	Start    time.Time
	End      time.Time
	Category string
}

type DayJSON struct {
	Date  string `json:"date"`
	Total string `json:"total"`
}

type CategoryJSON struct {
	Date     string `json:"date"`
	Category string `json:"category"`
	Total    string `json:"total"`
}
