package bot

import (
    "fmt"
    "io"
    
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
    api            *tgbotapi.BotAPI
    targetChatID   int64
    storageChatID  int64
}

func New(token string, targetChatID, storageChatID int64) (*Bot, error) {
    api, err := tgbotapi.NewBotAPI(token)
    if err != nil {
        return nil, err
    }
    
    return &Bot{
        api:           api,
        targetChatID:  targetChatID,
        storageChatID: storageChatID,
    }, nil
}

// SendDailyMessage sends the message to the main group
func (b *Bot) SendDailyMessage(text, fileID string) error {
    if fileID != "" {
        photo := tgbotapi.NewPhoto(b.targetChatID, tgbotapi.FileID(fileID))
        photo.Caption = text
        photo.ParseMode = "HTML"
        _, err := b.api.Send(photo)
        return err
    }
    
    msg := tgbotapi.NewMessage(b.targetChatID, text)
    msg.ParseMode = "HTML"
    _, err := b.api.Send(msg)
    return err
}

// UploadImage stores an image in the storage channel and returns its file_id
func (b *Bot) UploadImage(imageData io.Reader, filename string) (string, error) {
    if b.storageChatID == 0 {
        return "", fmt.Errorf("storage chat ID not configured")
    }
    
    // Read into buffer for potential retry
    data, err := io.ReadAll(imageData)
    if err != nil {
        return "", err
    }
    
    photo := tgbotapi.NewPhoto(b.storageChatID, tgbotapi.FileBytes{
        Name:  filename,
        Bytes: data,
    })
    
    msg, err := b.api.Send(photo)
    if err != nil {
        return "", err
    }
    
    // Get the highest resolution file_id
    if len(msg.Photo) > 0 {
        return msg.Photo[len(msg.Photo)-1].FileID, nil
    }
    
    return "", fmt.Errorf("no photo in response")
}