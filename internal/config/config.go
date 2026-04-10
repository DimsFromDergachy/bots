package config

import (
    "fmt"
    "os"
    "strconv"
)

type Config struct {
    // Telegram
    TelegramToken string
    TargetChatID  int64
    StorageChatID int64 // Private channel for image storage

    // Admin Panel
    AdminPort     string
    AdminUser     string
    AdminPass     string
    SessionSecret string

    // Scheduler
    Timezone   string
    SendHour   int
    SendMinute int

    // DB
    DBPath string
}

func Load() (*Config, error) {
    // Default values
    cfg := &Config{
        AdminPort:     "8080",
        Timezone:      "Europe/Moscow",
        SendHour:      9,
        SendMinute:    0,
        DBPath:        "./data/bible.db",
    }

    // Override from env
    if v := os.Getenv("TELEGRAM_BOT_TOKEN"); v != "" {
        cfg.TelegramToken = v
    }
    if v := os.Getenv("TELEGRAM_TARGET_CHAT_ID"); v != "" {
        id, _ := strconv.ParseInt(v, 10, 64)
        cfg.TargetChatID = id
    }
    if v := os.Getenv("TELEGRAM_STORAGE_CHAT_ID"); v != "" {
        id, _ := strconv.ParseInt(v, 10, 64)
        cfg.StorageChatID = id
    }
    if v := os.Getenv("ADMIN_PORT"); v != "" {
        cfg.AdminPort = v
    }
    if v := os.Getenv("ADMIN_USER"); v != "" {
        cfg.AdminUser = v
    }
    if v := os.Getenv("ADMIN_PASS"); v != "" {
        cfg.AdminPass = v
    }
    if v := os.Getenv("SESSION_SECRET"); v != "" {
        cfg.SessionSecret = v
    }
    if v := os.Getenv("TZ"); v != "" {
        cfg.Timezone = v
    }
    if v := os.Getenv("SEND_HOUR"); v != "" {
        h, _ := strconv.Atoi(v)
        cfg.SendHour = h
    }
    if v := os.Getenv("DB_PATH"); v != "" {
        cfg.DBPath = v
    }

    // Validate required
    if cfg.TelegramToken == "" {
        return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
    }
    if cfg.TargetChatID == 0 {
        return nil, fmt.Errorf("TELEGRAM_TARGET_CHAT_ID is required")
    }

    // Validate security stuff
    if cfg.AdminPass == "" {
        return nil, fmt.Errorf("ADMIN_PASS is required")
    }
    if cfg.SessionSecret == "" {
        return nil, fmt.Errorf("SESSION_SECRET is required")
    }

    return cfg, nil
}
