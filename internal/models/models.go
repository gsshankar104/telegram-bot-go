package models

import "time"

// User represents a Telegram user in the system
type User struct {
    UserID     int64     `json:"user_id"`
    Username   string    `json:"username"`
    FirstName  string    `json:"first_name"`
    LastName   string    `json:"last_name"`
    JoinedDate time.Time `json:"joined_date"`
    Status     string    `json:"status"` // active/blocked
}

// LotteryEntry represents a single lottery ticket entry
type LotteryEntry struct {
    EntryID      string    `json:"entry_id"`
    UserID       int64     `json:"user_id"`
    TicketAmount float64   `json:"ticket_amount"`
    TransactionID string    `json:"transaction_id"`
    UniqueCode   string    `json:"unique_code"`
    LuckyNumber  int       `json:"lucky_number"`
    EntryDate    time.Time `json:"entry_date"`
    EntryTime    time.Time `json:"entry_time"`
    Status       string    `json:"status"` // active/winner/expired
}

// Transaction represents a payment transaction
type Transaction struct {
    TransactionID string    `json:"transaction_id"`
    UserID        int64     `json:"user_id"`
    Amount        float64   `json:"amount"`
    Date          time.Time `json:"date"`
    Time          time.Time `json:"time"`
    Status        string    `json:"status"` // pending/verified/rejected
}

// Winner represents a lottery winner
type Winner struct {
    WinnerID            string    `json:"winner_id"`
    UserID              int64     `json:"user_id"`
    EntryID             string    `json:"entry_id"`
    WinningAmount       float64   `json:"winning_amount"`
    Date                time.Time `json:"date"`
    Time                time.Time `json:"time"`
    PaymentStatus       string    `json:"payment_status"` // pending/completed
    PaymentTransactionID string    `json:"payment_transaction_id"`
}

// UserState represents the current state of a user in the bot workflow
type UserState struct {
    UserID           int64     `json:"user_id"`
    CurrentState     string    `json:"current_state"`
    SelectedAmount   float64   `json:"selected_amount,omitempty"`
    TransactionID    string    `json:"transaction_id,omitempty"`
    UniqueCode       string    `json:"unique_code,omitempty"`
    InvalidAttempts  int       `json:"invalid_attempts"`
    LastUpdated      time.Time `json:"last_updated"`
}

// AdminAction represents an administrative action in the system
type AdminAction struct {
    ActionID    string    `json:"action_id"`
    AdminID     int64     `json:"admin_id"`
    ActionType  string    `json:"action_type"`
    Details     string    `json:"details"`
    Timestamp   time.Time `json:"timestamp"`
}

// Statistics represents system statistics
type Statistics struct {
    TotalUsers        int     `json:"total_users"`
    DailySales        float64 `json:"daily_sales"`
    WeeklySales       float64 `json:"weekly_sales"`
    MonthlySales      float64 `json:"monthly_sales"`
    PopularNumbers    []int   `json:"popular_numbers"`
    ActiveUsers       int     `json:"active_users"`
    LastUpdated       time.Time `json:"last_updated"`
}