package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

const schema = `
CREATE TABLE IF NOT EXISTS sessions (
    id INT AUTO_INCREMENT PRIMARY KEY,
    start_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    end_time DATETIME DEFAULT NULL,
    category VARCHAR(75) NOT NULL DEFAULT 'uncategorized',
    INDEX idx_start_time (start_time),
    INDEX idx_end_time (end_time)
)`

func mysqlDSN(cfg Config) string {
	c := mysql.Config{
		User:                 cfg.DBUser,
		Passwd:               cfg.DBPassword,
		Net:                  "tcp",
		Addr:                 cfg.DBHost,
		DBName:               cfg.DBName,
		AllowNativePasswords: true,
		ParseTime:            true,
		Loc:                  time.UTC,
	}
	return c.FormatDSN()
}

func openServer(cfg Config) (*sql.DB, error) {
	c := mysql.Config{
		User:                 cfg.DBUser,
		Passwd:               cfg.DBPassword,
		Net:                  "tcp",
		Addr:                 cfg.DBHost,
		AllowNativePasswords: true,
	}
	return sql.Open("mysql", c.FormatDSN())
}

func initDB(cfg Config) error {
	if cfg.DBName == "" {
		return fmt.Errorf("DB_NAME must be set")
	}
	db, err := openServer(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	if _, err := db.Exec("CREATE DATABASE IF NOT EXISTS " + quoteIdent(cfg.DBName)); err != nil {
		return err
	}
	if _, err := db.Exec("USE " + quoteIdent(cfg.DBName)); err != nil {
		return err
	}
	if _, err := db.Exec(schema); err != nil {
		return err
	}
	return nil
}

func quoteIdent(name string) string {
	return "`" + name + "`"
}

func getDB(cfg Config) (*sql.DB, error) {
	return sql.Open("mysql", mysqlDSN(cfg))
}

func getSession(db *sql.DB) (*SessionRow, error) {
	row := db.QueryRow(
		"SELECT id, start_time, end_time, category FROM sessions WHERE end_time IS NULL ORDER BY start_time DESC LIMIT 1",
	)
	s := &SessionRow{}
	err := row.Scan(&s.ID, &s.StartTime, &s.EndTime, &s.Category)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.StartTime = s.StartTime.Add(7 * time.Hour)
	return s, nil
}
