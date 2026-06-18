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
