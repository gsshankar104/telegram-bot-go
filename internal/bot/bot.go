package bot

import (
    "context"
    "fmt"
    "log"
    "strconv"
    "strings"
    "sync"
    "time"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    "github.com/user/telegram-bot/internal/config"
    "github.com/user/telegram-bot/internal/models"
    "github.com/user/telegram-bot/internal/storage"
)

type Bot struct {
    api         *tgbotapi.BotAPI
    storage     storage.Storage
    config      *config.Config
    rateLimiter *sync.Map
}

func New(storage storage.Storage, cfg *config.Config) (*Bot, error) {
    api, err := tgbotapi.NewBotAPI(cfg.Bot.Token)
    if err != nil {
        return nil, fmt.Errorf("failed to create bot: %v", err)
    }

    return &Bot{
        api:         api,
        storage:     storage,
        config:      cfg,
        rateLimiter: &sync.Map{},
    }, nil
}

func (b *Bot) Start(ctx context.Context) error {
    log.Printf("Starting %s", b.config.Bot.Name)

    updateConfig := tgbotapi.UpdateConfig{
        Timeout: 60,
    }
    updates := b.api.GetUpdatesChan(updateConfig)

    for {
        select {
        case update := <-updates:
            go b.handleUpdate(ctx, update)
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}

func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Recovered from panic in handleUpdate: %v", r)
        }
    }()

    if update.Message != nil {
        if !b.checkRateLimit(update.Message.From.ID) {
            b.sendMessage(update.Message.Chat.ID, "⚠️ आप बहुत तेजी से कमांड भेज रहे हैं। कृपया कुछ देर प्रतीक्षा करें।")
            return
        }

        switch {
        case update.Message.IsCommand():
            b.handleCommand(ctx, update.Message)
        case update.Message.Text != "":
            state, err := b.storage.GetUserState(ctx, update.Message.From.ID)
            if err != nil {
                log.Printf("Failed to get user state: %v", err)
                b.sendMessage(update.Message.Chat.ID, "⚠️ कुछ गड़बड़ी हुई। कृपया /start कमांड से शुरू करें।")
                return
            }
            b.handleMessageWithState(ctx, update.Message, state)
        }
    } else if update.CallbackQuery != nil {
        b.handleCallback(ctx, update.CallbackQuery)
    }
}

func (b *Bot) handleCommand(ctx context.Context, message *tgbotapi.Message) {
    switch message.Command() {
    case "start":
        b.handleStartCommand(ctx, message)
    case "view_data":
        b.handleViewDataCommand(ctx, message)
    case "select_winner":
        b.handleSelectWinnerCommand(ctx, message)
    case "export_all_users":
        b.handleExportUsersCommand(ctx, message)
    default:
        b.sendMessage(message.Chat.ID, "⚠️ अमान्य कमांड")
    }
}

func (b *Bot) handleMessageWithState(ctx context.Context, message *tgbotapi.Message, state *models.UserState) {
    switch state.CurrentState {
    case "awaiting_transaction_id":
        b.handleTransactionIDSubmission(ctx, message, state)
    case "awaiting_lucky_number":
        b.handleLuckyNumberSubmission(ctx, message, state)
    case "selecting_winner_count":
        b.handleWinnerCountSubmission(ctx, message, state)
    default:
        b.handleUnexpectedInput(ctx, message, state)
    }
}

func (b *Bot) handleCallback(ctx context.Context, callback *tgbotapi.CallbackQuery) {
    parts := strings.Split(callback.Data, ":")
    if len(parts) < 2 {
        return
    }

    action := parts[0]
    data := parts[1]

    switch action {
    case "select_amount":
        b.handleAmountSelection(ctx, callback, data)
    case "navigation":
        b.handleNavigation(ctx, callback, data)
    case "winner_amount":
        b.handleWinnerAmountSelection(ctx, callback, data)
    }

    callbackConfig := tgbotapi.NewCallback(callback.ID, "")
    b.api.Send(callbackConfig)
}

func (b *Bot) handleStartCommand(ctx context.Context, message *tgbotapi.Message) {
    user := &models.User{
        UserID:     message.From.ID,
        Username:   message.From.UserName,
        FirstName:  message.From.FirstName,
        LastName:   message.From.LastName,
        JoinedDate: time.Now(),
        Status:     "active",
    }

    if err := b.storage.SaveUser(ctx, user); err != nil {
        log.Printf("Failed to save user: %v", err)
    }

    welcomeMsg := fmt.Sprintf(
        "नमस्ते %s, %s में आपका स्वागत है! 👋\n\n"+
            "यह एक Lottery Bot है, टिकट खरीदने और भाग्य आजमाने के लिए नीचे दिए गए बटन पर क्लिक करें।\n\n"+
            "लॉटरी प्रूफ चैनल: %s",
        message.From.FirstName,
        b.config.Bot.Name,
        b.config.Channels.LotteryProof,
    )

    buttons := [][]string{
        {"लॉटरी टिकट खरीदें|select_amount:start"},
    }

    keyboard := b.createInlineKeyboard(buttons)
    msg := tgbotapi.NewMessage(message.Chat.ID, welcomeMsg)
    msg.ReplyMarkup = keyboard
    msg.ParseMode = "HTML"
    b.api.Send(msg)
}

func (b *Bot) handleTransactionIDSubmission(ctx context.Context, message *tgbotapi.Message, state *models.UserState) {
    txnID := message.Text

    used, err := b.storage.IsTransactionUsed(ctx, txnID)
    if err != nil {
        log.Printf("Failed to check transaction: %v", err)
        b.sendMessage(message.Chat.ID, "⚠️ कुछ गड़बड़ी हुई। कृपया पुनः प्रयास करें।")
        return
    }

    if used {
        b.sendMessage(message.Chat.ID, "⚠️ माफ़ करना! यह Transaction ID पहले ही इस्तेमाल हो चुकी है। कृपया दूसरी Transaction ID डालें।")
        return
    }

    uniqueCode := fmt.Sprintf("LC%d", time.Now().UnixNano())
    state.CurrentState = "awaiting_lucky_number"
    state.TransactionID = txnID
    state.UniqueCode = uniqueCode

    if err := b.storage.SaveUserState(ctx, state); err != nil {
        log.Printf("Failed to update user state: %v", err)
        b.sendMessage(message.Chat.ID, "⚠️ कुछ गड़बड़ी हुई। कृपया पुनः प्रयास करें।")
        return
    }

    msg := fmt.Sprintf(
        "✅ पेमेंट कन्फर्म! आपका Transaction ID है: %s \n\n"+
            "आपका Unique Code है: %s \n\n"+
            "अब 1 से 100 के बीच कोई भी एक Lucky Number चुनें:",
        txnID,
        uniqueCode,
    )

    buttons := [][]string{
        {"पिछला|navigation:back", "होम|navigation:home"},
    }

    keyboard := b.createInlineKeyboard(buttons)
    msgConfig := tgbotapi.NewMessage(message.Chat.ID, msg)
    msgConfig.ReplyMarkup = keyboard
    b.api.Send(msgConfig)
}

func (b *Bot) handleLuckyNumberSubmission(ctx context.Context, message *tgbotapi.Message, state *models.UserState) {
    number, err := strconv.Atoi(message.Text)
    if err != nil || number < 1 || number > 100 {
        b.sendMessage(message.Chat.ID, "माफ़ करना, wrong number. कृपया 1 से 100 के बीच में ही नंबर चुनें।")
        return
    }

    entry := &models.LotteryEntry{
        EntryID:      fmt.Sprintf("ENTRY%d", time.Now().UnixNano()),
        UserID:       message.From.ID,
        TicketAmount: state.SelectedAmount,
        TransactionID: state.TransactionID,
        UniqueCode:   state.UniqueCode,
        LuckyNumber:  number,
        EntryDate:    time.Now(),
        EntryTime:    time.Now(),
        Status:       "active",
    }

    if err := b.storage.SaveLotteryEntry(ctx, entry); err != nil {
        log.Printf("Failed to save lottery entry: %v", err)
        b.sendMessage(message.Chat.ID, "⚠️ कुछ गड़बड़ी हुई। कृपया पुनः प्रयास करें।")
        return
    }

    b.storage.DeleteUserState(ctx, message.From.ID)

    msg := fmt.Sprintf(
        "👍 आपका नंबर %d चुना गया है! आपकी Lottery Entry सफलतापूर्वक रजिस्टर हो गई है। \n\n"+
            "Result [Lottery Result Announcement Time] पर Lottery Proof चैनल %s में announce किए जाएंगे। \n\n"+
            "शुभकामनाएं!",
        number,
        b.config.Channels.LotteryProof,
    )

    buttons := [][]string{
        {"होम|navigation:home"},
    }

    keyboard := b.createInlineKeyboard(buttons)
    msgConfig := tgbotapi.NewMessage(message.Chat.ID, msg)
    msgConfig.ReplyMarkup = keyboard
    b.api.Send(msgConfig)
}

func (b *Bot) handleWinnerCountSubmission(ctx context.Context, message *tgbotapi.Message, state *models.UserState) {
    count, err := strconv.Atoi(message.Text)
    if err != nil || count < 1 || count > 10 {
        b.sendMessage(message.Chat.ID, "⚠️ कृपया 1 से 10 के बीच एक नंबर भेजें")
        return
    }

    buttons := [][]string{
        {"Random|winner_method:random"},
        {"First Come First Serve|winner_method:fcfs"},
        {"Most Guessed Number|winner_method:most_guessed"},
        {"Least Guessed Number|winner_method:least_guessed"},
        {"Manual Selection|winner_method:manual"},
    }

    state.CurrentState = "selecting_winner_method"
    state.LastUpdated = time.Now()
    b.storage.SaveUserState(ctx, state)

    msg := fmt.Sprintf("Selection method क्या होगी? (Winners: %d)", count)
    keyboard := b.createInlineKeyboard(buttons)
    msgConfig := tgbotapi.NewMessage(message.Chat.ID, msg)
    msgConfig.ReplyMarkup = keyboard
    b.api.Send(msgConfig)
}

func (b *Bot) handleUnexpectedInput(ctx context.Context, message *tgbotapi.Message, state *models.UserState) {
    state.InvalidAttempts++
    if err := b.storage.SaveUserState(ctx, state); err != nil {
        log.Printf("Failed to update invalid attempts: %v", err)
    }

    if state.InvalidAttempts >= b.config.Limits.MaxInvalidAttempts {
        b.storage.DeleteUserState(ctx, message.From.ID)
        b.handleStartCommand(ctx, message)
        return
    }

    msg := fmt.Sprintf(
        "⚠️ माफ़ करना, मुझे आपका इनपुट समझ में नहीं आया!\n\n"+
            "कृपया दिए गए विकल्पों में से चुनें या सही format में input भेजें।\n\n"+
            "किसी भी समस्या के लिए, %s पर Lottery Win चैनल से संपर्क करें।",
        b.config.Channels.LotteryWin,
    )

    b.sendMessage(message.Chat.ID, msg)
}

func (b *Bot) handleViewDataCommand(ctx context.Context, message *tgbotapi.Message) {
    if !b.isAdmin(message.From.ID) {
        b.sendMessage(message.Chat.ID, "⚠️ आप Admin नहीं हैं!")
        return
    }

    args := strings.Fields(message.CommandArguments())
    if len(args) < 2 {
        b.sendViewDataHelp(message.Chat.ID)
        return
    }

    switch args[0] {
    case "user":
        b.handleViewUserData(ctx, message.Chat.ID, args[1])
    case "txn":
        b.handleViewTransactionData(ctx, message.Chat.ID, args[1])
    case "date":
        b.handleViewDateData(ctx, message.Chat.ID, args[1])
    case "txndate":
        b.handleViewTransactionsByDate(ctx, message.Chat.ID, args[1])
    case "help":
        b.sendViewDataHelp(message.Chat.ID)
    default:
        b.sendViewDataHelp(message.Chat.ID)
    }
}

func (b *Bot) handleSelectWinnerCommand(ctx context.Context, message *tgbotapi.Message) {
    if !b.isAdmin(message.From.ID) {
        b.sendMessage(message.Chat.ID, "⚠️ आप Admin नहीं हैं!")
        return
    }

    var buttons [][]string
    for _, price := range b.config.Tickets.Prices {
        priceStr := fmt.Sprintf("₹%.0f", price)
        buttons = append(buttons, []string{
            fmt.Sprintf("%s|winner_amount:%.0f", priceStr, price),
        })
    }

    msg := "कौनसा lottery amount का winner select करना है?"
    keyboard := b.createInlineKeyboard(buttons)
    msgConfig := tgbotapi.NewMessage(message.Chat.ID, msg)
    msgConfig.ReplyMarkup = keyboard
    b.api.Send(msgConfig)
}

func (b *Bot) handleExportUsersCommand(ctx context.Context, message *tgbotapi.Message) {
    if !b.isAdmin(message.From.ID) {
        b.sendMessage(message.Chat.ID, "⚠️ आप Admin नहीं हैं!")
        return
    }

    users, err := b.storage.GetAllUsers(ctx, time.Time{}, time.Now())
    if err != nil {
        log.Printf("Failed to get users: %v", err)
        b.sendMessage(message.Chat.ID, "⚠️ डेटा प्राप्त करने में त्रुटि हुई")
        return
    }

    var data string
    data = "User ID,Username,First Name,Last Name,Joined Date,Status\n"
    for _, user := range users {
        data += fmt.Sprintf("%d,%s,%s,%s,%s,%s\n",
            user.UserID,
            user.Username,
            user.FirstName,
            user.LastName,
            user.JoinedDate.Format("2006-01-02 15:04:05"),
            user.Status,
        )
    }

    doc := tgbotapi.NewDocument(message.Chat.ID, tgbotapi.FileBytes{
        Name:  "users.csv",
        Bytes: []byte(data),
    })
    _, err = b.api.Send(doc)
    if err != nil {
        log.Printf("Failed to send file: %v", err)
        b.sendMessage(message.Chat.ID, "⚠️ फ़ाइल भेजने में त्रुटि हुई")
    }
}

func (b *Bot) handleAmountSelection(ctx context.Context, callback *tgbotapi.CallbackQuery, data string) {
    if data == "start" {
        var buttons [][]string
        for _, price := range b.config.Tickets.Prices {
            priceStr := fmt.Sprintf("₹%.0f", price)
            buttons = append(buttons, []string{
                fmt.Sprintf("%s|select_amount:%.0f", priceStr, price),
            })
        }
        buttons = append(buttons, []string{"होम|navigation:home"})

        msg := "कृपया टिकट की कीमत चुनें:"
        edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, msg)
        keyboard := b.createInlineKeyboard(buttons)
        edit.ReplyMarkup = &keyboard
        b.api.Send(edit)
        return
    }

    amount, err := strconv.ParseFloat(data, 64)
    if err != nil {
        log.Printf("Invalid amount selection: %v", err)
        return
    }

    state := &models.UserState{
        UserID:         callback.From.ID,
        CurrentState:   "awaiting_transaction_id",
        SelectedAmount: amount,
        LastUpdated:    time.Now(),
    }

    if err := b.storage.SaveUserState(ctx, state); err != nil {
        log.Printf("Failed to save user state: %v", err)
        return
    }

    msg := fmt.Sprintf(
        "लॉटरी टिकट ₹%.0f के लिए पेमेंट करने के लिए नीचे दिए गए QR कोड का उपयोग करें: \n\n"+
            "पेमेंट करने के बाद, Transaction ID और पेमेंट का स्क्रीनशॉट भेजें।",
        amount,
    )

    buttons := [][]string{
        {"पिछला मेनू|navigation:back", "होम|navigation:home"},
    }

    keyboard := b.createInlineKeyboard(buttons)
    photo := tgbotapi.NewPhoto(callback.Message.Chat.ID, tgbotapi.FileURL(b.config.Payment.QRCodeLink))
    photo.Caption = msg
    photo.ReplyMarkup = keyboard
    b.api.Send(photo)
}

func (b *Bot) handleNavigation(ctx context.Context, callback *tgbotapi.CallbackQuery, action string) {
    switch action {
    case "home":
        b.handleStartCommand(ctx, callback.Message)
    case "back":
        if state, err := b.storage.GetUserState(ctx, callback.From.ID); err == nil {
            if state.CurrentState == "awaiting_transaction_id" {
                b.handleAmountSelection(ctx, callback, "start")
            }
        }
    }
}

func (b *Bot) handleWinnerAmountSelection(ctx context.Context, callback *tgbotapi.CallbackQuery, data string) {
    if !b.isAdmin(callback.From.ID) {
        b.sendMessage(callback.Message.Chat.ID, "⚠️ आप Admin नहीं हैं!")
        return
    }

    amount, err := strconv.ParseFloat(data, 64)
    if err != nil {
        log.Printf("Invalid amount selection: %v", err)
        return
    }

    state := &models.UserState{
        UserID:         callback.From.ID,
        CurrentState:   "selecting_winner_count",
        SelectedAmount: amount,
        LastUpdated:    time.Now(),
    }

    if err := b.storage.SaveUserState(ctx, state); err != nil {
        log.Printf("Failed to save admin state: %v", err)
        return
    }

    msg := "कितने लोग जीतेंगे? (1-10 के बीच एक नंबर भेजें)"
    edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, msg)
    b.api.Send(edit)
}

func (b *Bot) handleViewUserData(ctx context.Context, chatID int64, username string) {
    // Implementation for viewing user data
    // You would need to add a method to storage interface to search by username
    b.sendMessage(chatID, "⚠️ Feature under development")
}

func (b *Bot) handleViewTransactionData(ctx context.Context, chatID int64, txnID string) {
    txn, err := b.storage.GetTransaction(ctx, txnID)
    if err != nil {
        b.sendMessage(chatID, "⚠️ Transaction not found")
        return
    }

    msg := fmt.Sprintf(
        "Transaction Details:\n"+
            "ID: %s\n"+
            "User ID: %d\n"+
            "Amount: ₹%.2f\n"+
            "Date: %s\n"+
            "Time: %s\n"+
            "Status: %s",
        txn.TransactionID,
        txn.UserID,
        txn.Amount,
        txn.Date.Format("2006-01-02"),
        txn.Time.Format("15:04:05"),
        txn.Status,
    )

    b.sendMessage(chatID, msg)
}

func (b *Bot) handleViewDateData(ctx context.Context, chatID int64, dateStr string) {
    date, err := time.Parse("2006-01-02", dateStr)
    if err != nil {
        b.sendMessage(chatID, "⚠️ Invalid date format. Use YYYY-MM-DD")
        return
    }

    entries, err := b.storage.GetEntriesByDate(ctx, date)
    if err != nil {
        b.sendMessage(chatID, "⚠️ Failed to get entries")
        return
    }

    if len(entries) == 0 {
        b.sendMessage(chatID, "No entries found for this date")
        return
    }

    msg := fmt.Sprintf("Entries for %s:\n\n", dateStr)
    for _, entry := range entries {
        msg += fmt.Sprintf(
            "Entry ID: %s\n"+
                "User ID: %d\n"+
                "Amount: ₹%.2f\n"+
                "Number: %d\n"+
                "Status: %s\n\n",
            entry.EntryID,
            entry.UserID,
            entry.TicketAmount,
            entry.LuckyNumber,
            entry.Status,
        )
    }

    b.sendMessage(chatID, msg)
}

func (b *Bot) handleViewTransactionsByDate(ctx context.Context, chatID int64, dateStr string) {
    date, err := time.Parse("2006-01-02", dateStr)
    if err != nil {
        b.sendMessage(chatID, "⚠️ Invalid date format. Use YYYY-MM-DD")
        return
    }

    txns, err := b.storage.GetTransactionsByDate(ctx, date)
    if err != nil {
        b.sendMessage(chatID, "⚠️ Failed to get transactions")
        return
    }

    if len(txns) == 0 {
        b.sendMessage(chatID, "No transactions found for this date")
        return
    }

    msg := fmt.Sprintf("Transactions for %s:\n\n", dateStr)
    for _, txn := range txns {
        msg += fmt.Sprintf(
            "Transaction ID: %s\n"+
                "User ID: %d\n"+
                "Amount: ₹%.2f\n"+
                "Time: %s\n"+
                "Status: %s\n\n",
            txn.TransactionID,
            txn.UserID,
            txn.Amount,
            txn.Time.Format("15:04:05"),
            txn.Status,
        )
    }

    b.sendMessage(chatID, msg)
}

func (b *Bot) sendViewDataHelp(chatID int64) {
    help := `View Data Commands:
/view_data user @username - View user data
/view_data txn <transaction_id> - View transaction data
/view_data date YYYY-MM-DD - View lottery entries for date
/view_data txndate YYYY-MM-DD - View transactions for date
/view_data help - Show this help message`
    
    b.sendMessage(chatID, help)
}

func (b *Bot) checkRateLimit(userID int64) bool {
    key := strconv.FormatInt(userID, 10)
    now := time.Now()

    if val, ok := b.rateLimiter.Load(key); ok {
        lastTime := val.(time.Time)
        if now.Sub(lastTime).Seconds() < float64(60/b.config.Limits.CommandRateLimit) {
            return false
        }
    }

    b.rateLimiter.Store(key, now)
    return true
}

func (b *Bot) sendMessage(chatID int64, text string, opts ...interface{}) (tgbotapi.Message, error) {
    msg := tgbotapi.NewMessage(chatID, text)
    
    for _, opt := range opts {
        switch v := opt.(type) {
        case tgbotapi.InlineKeyboardMarkup:
            msg.ReplyMarkup = v
        case bool:
            msg.ParseMode = "HTML"
        }
    }

    return b.api.Send(msg)
}

func (b *Bot) createInlineKeyboard(buttons [][]string) tgbotapi.InlineKeyboardMarkup {
    var keyboard [][]tgbotapi.InlineKeyboardButton

    for _, row := range buttons {
        var keyboardRow []tgbotapi.InlineKeyboardButton
        for _, button := range row {
            parts := strings.Split(button, "|")
            text := parts[0]
            data := parts[1]
            keyboardRow = append(keyboardRow, tgbotapi.NewInlineKeyboardButtonData(text, data))
        }
        keyboard = append(keyboard, keyboardRow)
    }

    return tgbotapi.NewInlineKeyboardMarkup(keyboard...)
}

func (b *Bot) isAdmin(userID int64) bool {
    userIDStr := strconv.FormatInt(userID, 10)
    for _, adminID := range b.config.Admin.IDs {
        if adminID == userIDStr {
            return true
        }
    }
    return false
}