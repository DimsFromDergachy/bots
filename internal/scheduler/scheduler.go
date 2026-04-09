package scheduler

import (
    "context"
    "log"
    "strconv"
    "time"
    
    "github.com/DimsFromDergachy/bots/internal/bot"
    "github.com/DimsFromDergachy/bots/internal/db"
)

type Scheduler struct {
    db           *db.DB
    bot          *bot.Bot
    loc          *time.Location
    hour         int
    minute       int
    lastSettingsCheck time.Time
}

func New(db *db.DB, bot *bot.Bot) (*Scheduler, error) {
    s := &Scheduler{
        db:  db,
        bot: bot,
    }
    
    // Load initial settings
    if err := s.loadSettings(); err != nil {
        return nil, err
    }
    
    return s, nil
}

func (s *Scheduler) loadSettings() error {
    // Load timezone
    tz, err := s.db.GetSetting("timezone")
    if err != nil || tz == "" {
        tz = "Europe/Moscow"
    }
    
    loc, err := time.LoadLocation(tz)
    if err != nil {
        log.Printf("Invalid timezone %s, falling back to UTC", tz)
        loc = time.UTC
    }
    s.loc = loc
    
    // Load hour
    hourStr, _ := s.db.GetSetting("send_hour")
    if hourStr != "" {
        s.hour, _ = strconv.Atoi(hourStr)
    }
    if s.hour < 0 || s.hour > 23 {
        s.hour = 9
    }
    
    // Load minute
    minuteStr, _ := s.db.GetSetting("send_minute")
    if minuteStr != "" {
        s.minute, _ = strconv.Atoi(minuteStr)
    }
    if s.minute < 0 || s.minute > 59 {
        s.minute = 0
    }
    
    log.Printf("Settings loaded: timezone=%s, send_time=%02d:%02d", 
        s.loc.String(), s.hour, s.minute)
    
    return nil
}

func (s *Scheduler) Start(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Minute)
    settingsTicker := time.NewTicker(5 * time.Minute) // Reload settings every 5 min
    
    go func() {
        s.checkAndSend()
        
        for {
            select {
            case <-ticker.C:
                s.checkAndSend()
            case <-settingsTicker.C:
                s.loadSettings()
            case <-ctx.Done():
                ticker.Stop()
                settingsTicker.Stop()
                return
            }
        }
    }()
}

func (s *Scheduler) checkAndSend() {
    now := time.Now().In(s.loc)
    
    // Check if within 5-minute window of configured time
    if now.Hour() != s.hour {
        return
    }
    if now.Minute() < s.minute || now.Minute() >= s.minute+5 {
        return
    }
    
    month := int(now.Month())
    day := now.Day()
    year := now.Year()
    
    sent, err := s.db.WasSentToday(month, day, year)
    if err != nil {
        log.Printf("ERROR checking sent status: %v", err)
        return
    }
    if sent {
        return
    }
    
    msg, err := s.db.GetMessage(month, day)
    if err != nil {
        log.Printf("ERROR fetching message for %02d-%02d: %v", month, day, err)
        return
    }
    if msg == nil || msg.Text == "" {
        return
    }
    
    if err := s.bot.SendDailyMessage(msg.Text, msg.ImageFileID); err != nil {
        log.Printf("ERROR sending message for %02d-%02d: %v", month, day, err)
        return
    }
    
    s.db.MarkAsSent(month, day, year)
    log.Printf("SUCCESS: Sent daily message for %02d-%02d at %02d:%02d", 
        month, day, s.hour, s.minute)
}

// Public method to force settings reload
func (s *Scheduler) ReloadSettings() error {
    return s.loadSettings()
}

// Getters for current settings
func (s *Scheduler) GetCurrentHour() int {
    return s.hour
}

func (s *Scheduler) GetCurrentMinute() int {
    return s.minute
}

func (s *Scheduler) GetCurrentTimezone() string {
    return s.loc.String()
}