package admin

import (
    "fmt"
    "html/template"
    "net/http"
    "strconv"
    
    "github.com/go-chi/chi/v5"
    "github.com/gorilla/sessions"
    "golang.org/x/crypto/bcrypt"
    
    "github.com/DimsFromDergachy/bots/internal/bot"
    "github.com/DimsFromDergachy/bots/internal/db"
)

type Admin struct {
    db      *db.DB
    bot     *bot.Bot
    store   *sessions.CookieStore
    cache   map[string]*template.Template
    username string
    password string
}

func New(db *db.DB, bot *bot.Bot, sessionSecret, username, password string) (*Admin, error) {
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
    })
    
    return r
}

func (a *Admin) AuthMiddleware(next http.Handler) http.Handler {
    fmt.Printf("AuthMiddleware 1")

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        session, err := a.store.Get(r, "bible-bot")

        if err != nil {
            fmt.Printf("Failed to save session: %v\n", err)
            http.Error(w, "Session error", http.StatusInternalServerError)
            return
        }

        if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
            fmt.Printf("Auth check - auth: %v, ok: %v, path: %s\n", auth, ok, r.URL.Path)
            http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
            fmt.Printf("AuthMiddleware 2")
            return
        }
        fmt.Printf("Authenticated, serving content\n")
        fmt.Printf("AuthMiddleware 3")
        next.ServeHTTP(w, r)
    })
}

func (a *Admin) LoginPage(w http.ResponseWriter, r *http.Request) {
            fmt.Printf("AuthMiddleware 4")
    a.cache["login.html"].ExecuteTemplate(w, "base.html", nil)
}

func (a *Admin) Login(w http.ResponseWriter, r *http.Request) {
            fmt.Printf("AuthMiddleware 5")
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
            fmt.Printf("AuthMiddleware 6")
    session, _ := a.store.Get(r, "bible-bot")
    session.Values["authenticated"] = false
    session.Save(r, w)
    http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func (a *Admin) Calendar(w http.ResponseWriter, r *http.Request) {
    session, _ := a.store.Get(r, "bible-bot")
    fmt.Printf("Calendar handler - Session values: %+v\n", session.Values)
    
    fmt.Printf("AuthMiddleware 7")

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

    err = a.cache["calendar.html"].ExecuteTemplate(w, "base.html", data)
}

func (a *Admin) EditPage(w http.ResponseWriter, r *http.Request) {
            fmt.Printf("AuthMiddleware 8")
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
    
    a.cache["edit.html"].ExecuteTemplate(w, "base.html", data)
}

func (a *Admin) SaveMessage(w http.ResponseWriter, r *http.Request) {
            fmt.Printf("AuthMiddleware 9")
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