package db

import (
    "database/sql"
    "fmt"
)

var migrations = []string{
    `CREATE TABLE IF NOT EXISTS daily_messages (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        month INTEGER NOT NULL CHECK (month BETWEEN 1 AND 12),
        day INTEGER NOT NULL CHECK (day BETWEEN 1 AND 31),
        text TEXT NOT NULL DEFAULT '',
        image_file_id TEXT DEFAULT '',
        image_url TEXT DEFAULT '',
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(month, day)
    )`,
    
    `CREATE TABLE IF NOT EXISTS sent_log (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        month INTEGER NOT NULL,
        day INTEGER NOT NULL,
        year INTEGER NOT NULL,
        sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(month, day, year)
    )`,
    
    `CREATE INDEX IF NOT EXISTS idx_sent_log_date ON sent_log(month, day, year)`,
    
    `CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        username TEXT UNIQUE NOT NULL,
        password_hash TEXT NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    )`,
    
    // Pre-populate all 366 days with empty messages
    `INSERT OR IGNORE INTO daily_messages (month, day) 
     WITH RECURSIVE
     months(m) AS (SELECT 1 UNION ALL SELECT m+1 FROM months WHERE m<12),
     days(d) AS (SELECT 1 UNION ALL SELECT d+1 FROM days WHERE d<31)
     SELECT m, d FROM months, days 
     WHERE (m=2 AND d<=29) OR (m IN (4,6,9,11) AND d<=30) OR (m IN (1,3,5,7,8,10,12) AND d<=31)
     ORDER BY m, d`,

    // Add settings table
    `CREATE TABLE IF NOT EXISTS settings (
        key TEXT PRIMARY KEY,
        value TEXT NOT NULL,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
    )`,
    
    // Insert default settings
    `INSERT OR IGNORE INTO settings (key, value) VALUES 
        ('send_hour', '9'),
        ('send_minute', '0'),
        ('timezone', 'Europe/Moscow')`,
}

func RunMigrations(db *sql.DB) error {
    for i, migration := range migrations {
        _, err := db.Exec(migration)
        if err != nil {
            return fmt.Errorf("migration %d failed: %w", i, err)
        }
    }
    return nil
}