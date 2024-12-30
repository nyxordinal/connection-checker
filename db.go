package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/glebarez/sqlite"
)

type Database struct {
	db *sql.DB
}

func initDB() (*Database, error) {
	dbl, err := sql.Open("sqlite", "./logs.db")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	_, err = dbl.Exec(`CREATE TABLE IF NOT EXISTS connection_status (
		id INTEGER PRIMARY KEY,
		status TEXT NOT NULL,
		last_email_sent DATETIME
	);`)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection_status table: %w", err)
	}

	_, err = dbl.Exec(`INSERT OR IGNORE INTO connection_status (id, status, last_email_sent)
		VALUES (1, 'unknown', '1970-01-01 12:00:00');`)
	if err != nil {
		return nil, fmt.Errorf("failed to insert initial connection status: %w", err)
	}

	_, err = dbl.Exec(`CREATE TABLE IF NOT EXISTS logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp TEXT,
		status TEXT
	);`)
	if err != nil {
		return nil, fmt.Errorf("failed to create logs table: %w", err)
	}

	_, err = dbl.Exec(`CREATE INDEX IF NOT EXISTS idx_logs_status ON logs (status);`)
	if err != nil {
		return nil, fmt.Errorf("failed to create index on logs table: %w", err)
	}

	_, err = dbl.Exec(`CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs (timestamp);`)
	if err != nil {
		return nil, fmt.Errorf("failed to create index on logs table: %w", err)
	}

	return &Database{db: dbl}, nil
}

func (d *Database) getConnectionStatus() (string, string, error) {
	var status, lastEmailSent string
	row := d.db.QueryRow("SELECT status, last_email_sent FROM connection_status WHERE id = 1;")
	err := row.Scan(&status, &lastEmailSent)
	if err != nil {
		return "", "", err
	}
	return status, lastEmailSent, nil
}

func (d *Database) updateConnectionStatus(status string) error {
	_, err := d.db.Exec("UPDATE connection_status SET status = ? WHERE id = 1;", status)
	return err
}

func (d *Database) updateLastSentEmail(timestamp string) error {
	_, err := d.db.Exec("UPDATE connection_status SET last_email_sent = ? WHERE id = 1;", timestamp)
	return err
}

func (d *Database) getConnectionLogs(page, perPage int) ([]map[string]interface{}, error) {
	rows, err := d.db.Query(`SELECT id, timestamp, status FROM logs ORDER BY timestamp DESC LIMIT ? OFFSET ?`, perPage, (page-1)*perPage)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve logs: %v", err)
	}
	defer rows.Close()

	logs := []map[string]interface{}{}
	for rows.Next() {
		var id int
		var timestamp, status string
		if err := rows.Scan(&id, &timestamp, &status); err != nil {
			return nil, fmt.Errorf("failed to read log entry: %v", err)
		}

		logs = append(logs, map[string]interface{}{
			"id":        id,
			"timestamp": timestamp,
			"status":    status,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over logs: %v", err)
	}

	return logs, nil
}

func (d *Database) logConnectionStatus(status string) error {
	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05 MST")
	_, err := d.db.Exec("INSERT INTO logs (timestamp, status) VALUES (?, ?)", timestamp, status)
	return err
}
