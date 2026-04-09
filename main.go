package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/joho/godotenv"
    
    "github.com/DimsFromDergachy/bots/internal/admin"
    "github.com/DimsFromDergachy/bots/internal/bot"
    "github.com/DimsFromDergachy/bots/internal/config"
    "github.com/DimsFromDergachy/bots/internal/db"
    "github.com/DimsFromDergachy/bots/internal/scheduler"
)

func main() {
    // Load .env file if exists
    godotenv.Load()
    
    // Load config
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Config error: %v", err)
    }
    
    // Initialize database
    database, err := db.New(cfg.DBPath)
    if err != nil {
        log.Fatalf("Database error: %v", err)
    }
    defer database.Close()
    
    // Initialize Telegram bot
    telegramBot, err := bot.New(cfg.TelegramToken, cfg.TargetChatID, cfg.StorageChatID)
    if err != nil {
        log.Fatalf("Telegram bot error: %v", err)
    }
    
    // Initialize scheduler (simplified - no config needed)
    sched, err := scheduler.New(database, telegramBot)
    if err != nil {
        log.Fatalf("Scheduler error: %v", err)
    }

    // Context for graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Start scheduler
    sched.Start(ctx)
    log.Printf("Scheduler started. Will send daily at %02d:%02d %s", cfg.SendHour, cfg.SendMinute, cfg.Timezone)
    
    // Initialize admin panel
    adminPanel, err := admin.New(database, telegramBot, sched, cfg.SessionSecret, cfg.AdminUser, cfg.AdminPass)
    if err != nil {
        log.Fatalf("Admin panel error: %v", err)
    }
    
    // Setup HTTP router
    r := chi.NewRouter()
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(middleware.Compress(5))
    
    // Mount admin routes
    r.Mount("/admin", adminPanel.Routes())
    
    // Serve static files (if any)
    r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
    
    // Health check
    r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("OK"))
    })
    
    // Start HTTP server
    server := &http.Server{
        Addr:    ":" + cfg.AdminPort,
        Handler: r,
    }
    
    go func() {
        log.Printf("Admin panel listening on http://localhost:%s", cfg.AdminPort)
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("HTTP server error: %v", err)
        }
    }()
    
    // Wait for interrupt signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan
    
    log.Println("Shutting down...")
    
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer shutdownCancel()
    
    server.Shutdown(shutdownCtx)
    cancel()
    
    log.Println("Shutdown complete")
}