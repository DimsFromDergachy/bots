package scheduler

import (
    "context"
    "log"
    "time"
    
    "github.com/DimsFromDergachy/bots/internal/bot"
    "github.com/DimsFromDergachy/bots/internal/db"
)

type Scheduler struct {
    db     *db.DB
    bot    *bot.Bot
    loc    *time.Location
    hour   int
    minute int
}

func New(db *db.DB, bot *bot.Bot, timezone string, hour, minute int) (*Scheduler, error) {
    loc, err := time.LoadLocation(timezone)
    if err != nil {
        return nil, err
    }
    
    return &Scheduler{
        db:     db,
        bot:    bot,
        loc:    loc,
        hour:   hour,
        minute: minute,
    }, nil
}

func (s *Scheduler) Start(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Minute)
    
    go func() {
        // Run immediately on startup (catch up if we were down)
        s.checkAndSend()
        
        for {
            select {
            case <-ticker.C:
                s.checkAndSend()
            case <-ctx.Done():
                ticker.Stop()
                return
            }
        }
    }()
}

func (s *Scheduler) checkAndSend() {
    now := time.Now().In(s.loc)
    
    // Only send within a 5-minute window
    if now.Hour() != s.hour || now.Minute() < s.minute || now.Minute() >= s.minute+5 {
        return
    }
    
    month := int(now.Month())
    day := now.Day()
    year := now.Year()
    
    // Check if already sent today
    sent, err := s.db.WasSentToday(month, day, year)
    if err != nil {
        log.Printf("ERROR checking sent status: %v", err)
        return
    }
    if sent {
        return
    }
    
    // Get message
    msg, err := s.db.GetMessage(month, day)
    if err != nil {
        log.Printf("ERROR fetching message for %02d-%02d: %v", month, day, err)
        return
    }
    if msg == nil || msg.Text == "" {
        log.Printf("No message configured for %02d-%02d", month, day)
        return
    }
    
    // Send
    if err := s.bot.SendDailyMessage(msg.Text, msg.ImageFileID); err != nil {
        log.Printf("ERROR sending message for %02d-%02d: %v", month, day, err)
        return
    }
    
    // Mark as sent
    if err := s.db.MarkAsSent(month, day, year); err != nil {
        log.Printf("ERROR marking as sent: %v", err)
    }
    
    log.Printf("SUCCESS: Sent daily message for %02d-%02d", month, day)
}