# Bible Bot - Telegram Daily Message Scheduler

## Overview
A Go application that automatically sends daily Bible reading messages to a Telegram channel. Includes a web-based admin panel for managing the content.

**Stack:** Go 1.22+, SQLite, Docker, Telegram Bot API

## Core Features
- **Daily Scheduler**: Sends pre-configured messages (text + image) to a Telegram channel at a specific time each day
- **Idempotent Delivery**: Uses `sent_log` table to ensure messages are never sent twice for the same date
- **Admin Panel**: Web UI at `/admin` for editing all 366 daily messages
- **Image Storage**: Images are cached in a private Telegram channel; only `file_id` is stored in SQLite

## Project Structure
```
.
├── main.go                # Entry point, orchestrates all components
├── internal/
│   ├── bot/               # Telegram API wrapper
│   ├── scheduler/         # Daily cron-like job with timezone awareness
│   ├── admin/             # Web handlers + HTML templates
│   ├── db/                # SQLite migrations and queries
│   └── config/            # Environment configuration
├── templates/             # Go HTML templates for admin UI
├── data/                  # SQLite database (mounted volume in Docker)
├── Dockerfile
└── docker-compose.yml
```

## Key Implementation Details

### Scheduler Logic
- Runs every minute, checks if current time matches `SEND_HOUR:00-05` window
- Queries `sent_log` before sending to prevent duplicates
- Timezone-aware via `TZ` environment variable (e.g., `Europe/Moscow`)

### Database Schema
```sql
daily_messages (month, day, text, image_file_id, updated_at)  -- 366 pre-populated rows
sent_log (month, day, year, sent_at)                          -- Tracks sent status
users (username, password_hash)                               -- Admin authentication
```

### Image Upload Flow
1. Admin uploads image via web form
2. Backend sends to private Telegram storage channel
3. Telegram returns `file_id` (permanent reference)
4. `file_id` saved to SQLite; actual image discarded

### Admin Panel
- Built with Go `html/template`, Tailwind CSS, and HTMX
- Session-based auth using `gorilla/sessions`
- Calendar grid view showing all months/days with edit links

## Environment Variables
| Variable | Required | Description |
|----------|----------|-------------|
| `TELEGRAM_BOT_TOKEN` | Yes | Bot token from @BotFather |
| `TELEGRAM_TARGET_CHAT_ID` | Yes | Target channel ID (negative number) |
| `TELEGRAM_STORAGE_CHAT_ID` | Yes | Private channel for image storage and test messages as well |
| `ADMIN_USER` | No | Admin username (default: `admin`) |
| `ADMIN_PASS` | No | Admin password (default: `changeme`) |
| `SESSION_SECRET` | No | Cookie encryption key |
| `TZ` | No | Timezone (default: `Europe/Moscow`) |
| `SEND_HOUR` | No | Hour to send messages (default: `9`) |

## Development Commands
```bash
# Run locally
go run main.go

# Build and run with Docker
docker compose up -d

# View logs
docker compose logs -f

# Reset database
rm -rf data/*.db && docker compose restart
```

## Telegram Setup
1. Create bot via @BotFather → get token
2. Add bot to target channel as admin with "Post Messages" permission
3. Create private storage channel, add bot as admin
4. Get chat IDs via `https://api.telegram.org/bot<TOKEN>/getUpdates`

## Notes
- SQLite uses WAL mode for concurrent access
- Bot runs scheduler and web server in single binary
- Templates embedded via `html/template` (not `embed.FS` in current version)
- Admin auth can be bypassed during development by modifying `AuthMiddleware`

## Common Issues
- **"Database error: unable to open database file"**: Create `./data` directory
- **Template not found**: Run from project root where `templates/` exists
- **Login redirect loop**: Check session secret is set and cookie is being saved
