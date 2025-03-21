package storage

import (
    "context"
    "time"

    "github.com/gsshankar104/telegram-bot/internal/models"
)

// StorageError represents a custom error type for storage operations
type StorageError struct {
    Operation string
    Err       error
}

func (e *StorageError) Error() string {
    return e.Operation + ": " + e.Err.Error()
}

// NewStorageError creates a new StorageError
func NewStorageError(operation string, err error) *StorageError {
    return &StorageError{
        Operation: operation,
        Err:       err,
    }
}

// Storage defines the interface for data persistence
type Storage interface {
    // User operations
    SaveUser(ctx context.Context, user *models.User) error
    GetUser(ctx context.Context, userID int64) (*models.User, error)
    GetAllUsers(ctx context.Context, fromDate, toDate time.Time) ([]*models.User, error)

    // Transaction operations
    SaveTransaction(ctx context.Context, txn *models.Transaction) error
    GetTransaction(ctx context.Context, txnID string) (*models.Transaction, error)
    GetTransactionsByDate(ctx context.Context, date time.Time) ([]*models.Transaction, error)
    IsTransactionUsed(ctx context.Context, txnID string) (bool, error)

    // Lottery entry operations
    SaveLotteryEntry(ctx context.Context, entry *models.LotteryEntry) error
    GetEntriesByDate(ctx context.Context, date time.Time) ([]*models.LotteryEntry, error)

    // User state operations
    SaveUserState(ctx context.Context, state *models.UserState) error
    GetUserState(ctx context.Context, userID int64) (*models.UserState, error)
    DeleteUserState(ctx context.Context, userID int64) error
}
