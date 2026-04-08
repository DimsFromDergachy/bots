package db

import (
    "database/sql"
    "time"
)

type DailyMessage struct {
    ID          int64
    Month       int
    Day         int
    Text        string
    ImageFileID string
    ImageURL    string
    UpdatedAt   time.Time
}

func (db *DB) GetMessage(month, day int) (*DailyMessage, error) {
    var msg DailyMessage
    err := db.QueryRow(`
        SELECT id, month, day, text, image_file_id, image_url, updated_at 
        FROM daily_messages 
        WHERE month=? AND day=?`,
        month, day,
    ).Scan(&msg.ID, &msg.Month, &msg.Day, &msg.Text, &msg.ImageFileID, &msg.ImageURL, &msg.UpdatedAt)
    
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    return &msg, nil
}

func (db *DB) UpdateMessage(month, day int, text, imageFileID, imageURL string) error {
    _, err := db.Exec(`
        UPDATE daily_messages 
        SET text=?, image_file_id=?, image_url=?, updated_at=CURRENT_TIMESTAMP
        WHERE month=? AND day=?`,
        text, imageFileID, imageURL, month, day,
    )
    return err
}

func (db *DB) GetAllMessages() ([]DailyMessage, error) {
    rows, err := db.Query(`
        SELECT month, day, text, image_file_id, updated_at 
        FROM daily_messages 
        ORDER BY month, day`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var messages []DailyMessage
    for rows.Next() {
        var msg DailyMessage
        err := rows.Scan(&msg.Month, &msg.Day, &msg.Text, &msg.ImageFileID, &msg.UpdatedAt)
        if err != nil {
            return nil, err
        }
        messages = append(messages, msg)
    }
    return messages, nil
}

func (db *DB) WasSentToday(month, day, year int) (bool, error) {
    var exists bool
    err := db.QueryRow(`
        SELECT EXISTS(SELECT 1 FROM sent_log WHERE month=? AND day=? AND year=?)`,
        month, day, year,
    ).Scan(&exists)
    return exists, err
}

func (db *DB) MarkAsSent(month, day, year int) error {
    _, err := db.Exec(`
        INSERT OR IGNORE INTO sent_log (month, day, year) VALUES (?, ?, ?)`,
        month, day, year,
    )
    return err
}