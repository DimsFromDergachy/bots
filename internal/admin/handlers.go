package admin

import (
    "fmt"
    "html/template"
    "net/http"
    "strconv"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/gorilla/sessions"
    "golang.org/x/crypto/bcrypt"

    "github.com/DimsFromDergachy/bots/internal/bot"
    "github.com/DimsFromDergachy/bots/internal/db"
    "github.com/DimsFromDergachy/bots/internal/scheduler"
)

type Admin struct {
    db          *db.DB
    bot         *bot.Bot
    scheduler   *scheduler.Scheduler
    store       *sessions.CookieStore
    cache       map[string]*template.Template
    username    string
    password    string
}

func New(db *db.DB, bot *bot.Bot, sched *scheduler.Scheduler, sessionSecret, username, password string) (*Admin, error) {
    // Create default admin if not exists
    if username != "" && password != "" {
        hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
        db.Exec("INSERT OR IGNORE INTO users (username, password_hash) VALUES (?, ?)", username, string(hash))
    }
    
    cache := map[string]*template.Template{}

    pages := []string{
        "login.html",
        "calendar.html",
        "edit.html",
        "settings.html",
    }

    for _, page := range pages {
        cache[page] = template.Must(template.ParseFiles("templates/base.html", "templates/" + page))
    }

    store := sessions.NewCookieStore([]byte(sessionSecret))
    store.Options = &sessions.Options{
        Path:     "/",
        MaxAge:   86400 * 7, // 7 days
        HttpOnly: true,
        Secure:   false, // Set to true if using HTTPS
        SameSite: http.SameSiteLaxMode,
    }

    return &Admin{
        db:        db,
        bot:       bot,
        scheduler: sched,
        store:     store,
        cache:     cache,
        username:  username,
        password:  password,
    }, nil
}

func (a *Admin) Routes() chi.Router {
    r := chi.NewRouter()
    
    // Public
    r.Get("/login", a.LoginPage)
    r.Post("/login", a.Login)
    r.Get("/logout", a.Logout)

    // Protected
    r.Group(func(r chi.Router) {
        r.Use(a.AuthMiddleware)
        r.Get("/", a.Calendar)
        r.Get("/edit/{month}/{day}", a.EditPage)
        r.Post("/edit/{month}/{day}", a.SaveMessage)
        r.Post("/upload/{month}/{day}", a.UploadImage)
        r.Get("/settings", a.SettingsPage)
        r.Post("/settings", a.SaveSettings)
        r.Post("/settings/test-send", a.TestSend)
    })
    
    return r
}

func (a *Admin) AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        session, _ := a.store.Get(r, "bible-bot")

        if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
            http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
            return
        }
        next.ServeHTTP(w, r)
    })
}

func (a *Admin) LoginPage(w http.ResponseWriter, r *http.Request) {
    a.cache["login.html"].ExecuteTemplate(w, "login.html", nil)
}

func (a *Admin) Login(w http.ResponseWriter, r *http.Request) {
    username := r.FormValue("username")
    password := r.FormValue("password")
    
    var hash string
    err := a.db.QueryRow("SELECT password_hash FROM users WHERE username=?", username).Scan(&hash)
    if err != nil {
        http.Error(w, "Invalid credentials", http.StatusUnauthorized)
        return
    }
    
    if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
        http.Error(w, "Invalid credentials", http.StatusUnauthorized)
        return
    }
    
    session, _ := a.store.Get(r, "bible-bot")
    session.Values["authenticated"] = true
    session.Save(r, w)
    
    http.Redirect(w, r, "/admin/", http.StatusSeeOther)
}

func (a *Admin) Logout(w http.ResponseWriter, r *http.Request) {
    session, _ := a.store.Get(r, "bible-bot")
    session.Values["authenticated"] = false
    session.Save(r, w)
    http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func (a *Admin) Calendar(w http.ResponseWriter, r *http.Request) {
    messages, err := a.db.GetAllMessages()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Group by month
    messagesByMonth := make(map[int][]db.DailyMessage)
    for _, msg := range messages {
        messagesByMonth[msg.Month] = append(messagesByMonth[msg.Month], msg)
    }
    
    monthNames := []string{
        "", 
        "January", "February", "March", "April", "May", "June",
        "July", "August", "September", "October", "November", "December",
    }
    
    monthNumbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
    
    data := struct {
        MessagesByMonth map[int][]db.DailyMessage
        MonthNames      []string
        MonthNumbers    []int
    }{
        MessagesByMonth: messagesByMonth,
        MonthNames:      monthNames,
        MonthNumbers:    monthNumbers,
    }

    err = a.cache["calendar.html"].ExecuteTemplate(w, "calendar.html", data)
}

func (a *Admin) EditPage(w http.ResponseWriter, r *http.Request) {
    month, _ := strconv.Atoi(chi.URLParam(r, "month"))
    day, _ := strconv.Atoi(chi.URLParam(r, "day"))
    
    msg, err := a.db.GetMessage(month, day)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    data := struct {
        Message *db.DailyMessage
        Month   int
        Day     int
    }{
        Message: msg,
        Month:   month,
        Day:     day,
    }
    
    a.cache["edit.html"].ExecuteTemplate(w, "edit.html", data)
}

func (a *Admin) SaveMessage(w http.ResponseWriter, r *http.Request) {
    month, _ := strconv.Atoi(chi.URLParam(r, "month"))
    day, _ := strconv.Atoi(chi.URLParam(r, "day"))
    
    text := r.FormValue("text")
    
    // Keep existing image
    msg, _ := a.db.GetMessage(month, day)
    fileID := ""
    imageURL := ""
    if msg != nil {
        fileID = msg.ImageFileID
        imageURL = msg.ImageURL
    }
    
    if err := a.db.UpdateMessage(month, day, text, fileID, imageURL); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    http.Redirect(w, r, "/admin/", http.StatusSeeOther)
}

func (a *Admin) UploadImage(w http.ResponseWriter, r *http.Request) {
    month, _ := strconv.Atoi(chi.URLParam(r, "month"))
    day, _ := strconv.Atoi(chi.URLParam(r, "day"))
    
    file, header, err := r.FormFile("image")
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    defer file.Close()
    
    // Upload to Telegram storage
    fileID, err := a.bot.UploadImage(file, header.Filename)
    if err != nil {
        http.Error(w, fmt.Sprintf("Upload failed: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Update message
    msg, _ := a.db.GetMessage(month, day)
    text := ""
    if msg != nil {
        text = msg.Text
    }
    
    if err := a.db.UpdateMessage(month, day, text, fileID, ""); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Return JSON for HTMX
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprintf(w, `{"file_id":"%s"}`, fileID)
}

// Add handler methods:
func (a *Admin) SettingsPage(w http.ResponseWriter, r *http.Request) {
    settings, err := a.db.GetSettings()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Default values if not set
    if settings["send_hour"] == "" {
        settings["send_hour"] = "9"
    }
    if settings["send_minute"] == "" {
        settings["send_minute"] = "0"
    }
    if settings["timezone"] == "" {
        settings["timezone"] = "Europe/Moscow"
    }
    
    // Get available timezones
    timezones := getCommonTimezones()
    
    data := struct {
        Settings  map[string]string
        Timezones []TimezoneOption
        CurrentTime time.Time
        NextSendTime string
        TargetChatID int64
        StorageChatID int64
    }{
        Settings:  settings,
        Timezones: timezones,
        CurrentTime: time.Now(),
        NextSendTime: "NAN",
        TargetChatID: a.bot.GetTargetChatID(),
        StorageChatID: a.bot.GetStorageChatID(),
    }
    
    a.cache["settings.html"].ExecuteTemplate(w, "settings.html", data)
}

func (a *Admin) SaveSettings(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    settings := map[string]string{
        "send_hour":   r.FormValue("send_hour"),
        "send_minute": r.FormValue("send_minute"),
        "timezone":    r.FormValue("timezone"),
    }
    
    // Validate
    hour, _ := strconv.Atoi(settings["send_hour"])
    if hour < 0 || hour > 23 {
        settings["send_hour"] = "9"
    }
    
    minute, _ := strconv.Atoi(settings["send_minute"])
    if minute < 0 || minute > 59 {
        settings["send_minute"] = "0"
    }
    
    if err := a.db.SetSettings(settings); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Reload scheduler settings
    if a.scheduler != nil {
        a.scheduler.ReloadSettings()
    }
    
    http.Redirect(w, r, "/admin/settings", http.StatusSeeOther)
}

type TimezoneOption struct {
    Value string
    Label string
}

func getCommonTimezones() []TimezoneOption {
    return []TimezoneOption{
        {"Europe/Moscow", "Moscow (UTC+3)"},
        {"Europe/London", "London (UTC+0/+1)"},
        {"Europe/Paris", "Paris (UTC+1/+2)"},
        {"Europe/Kiev", "Kyiv (UTC+2/+3)"},
        {"America/New_York", "New York (UTC-5/-4)"},
        {"America/Chicago", "Chicago (UTC-6/-5)"},
        {"America/Denver", "Denver (UTC-7/-6)"},
        {"America/Los_Angeles", "Los Angeles (UTC-8/-7)"},
        {"Asia/Jerusalem", "Jerusalem (UTC+2/+3)"},
        {"UTC", "UTC"},
    }
}

func (a *Admin) TestSend(w http.ResponseWriter, r *http.Request) {
    now := time.Now()
    month := int(now.Month())
    day := now.Day()
    
    msg, err := a.db.GetMessage(month, day)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        fmt.Fprintf(w, "<span class='text-red-600'>Error: %v</span>", err)
        return
    }
    
    if msg == nil || msg.Text == "" {
        fmt.Fprintf(w, "<span class='text-yellow-600'>No message configured for today</span>")
        return
    }
    
    if err := a.bot.SendTestMessage(msg.Text, msg.ImageFileID); err != nil {
        fmt.Fprintf(w, "<span class='text-red-600'>Failed to send: %v</span>", err)
        return
    }
    
    fmt.Fprintf(w, "<span class='text-green-600'>✓ Test message sent successfully!</span>")
}