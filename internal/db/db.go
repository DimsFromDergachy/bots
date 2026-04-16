package db

import (
    "database/sql"
    "time"
    
    _ "modernc.org/sqlite" // sqlite driver
)

type DB struct {
    *sql.DB
}

func New(path string) (*DB, error) {
    db, err := sql.Open("sqlite", path)
    if err != nil {
        return nil, err
    }
    
    db.SetMaxOpenConns(1) // SQLite works best with single writer
    db.SetConnMaxLifetime(time.Hour)
    
    if err := RunMigrations(db); err != nil {
        return nil, err
    }
    
    return &DB{db}, nil
}

func (db *DB) Close() error {
    return db.DB.Close()
}