package drive

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
    "sync"
    "time"

    "github.com/gsshankar104/telegram-bot/internal/config"
    "github.com/gsshankar104/telegram-bot/internal/models"
    "github.com/gsshankar104/telegram-bot/internal/storage"
    
    "google.golang.org/api/drive/v3"
    "google.golang.org/api/option"
)

type DriveStorage struct {
    service      *drive.Service
    folderID     string
    mutex        sync.RWMutex
    fileCache    map[string]string // filename to fileId mapping
}

const (
    usersFile       = "users.json"
    entriesFile     = "lottery_entries.json"
    transactionsFile = "transactions.json"
    winnersFile     = "winners.json"
    statesFile      = "user_states.json"
    statsFile       = "statistics.json"
    adminActionsFile = "admin_actions.json"
)

func NewDriveStorage(ctx context.Context, credentialsFile string) (*DriveStorage, error) {
    service, err := drive.NewService(ctx, option.WithCredentialsFile(credentialsFile))
    if err != nil {
        return nil, fmt.Errorf("failed to create Drive client: %v", err)
    }

    ds := &DriveStorage{
        service:   service,
        folderID:  config.Get().Database.DriveFolderID,
        fileCache: make(map[string]string),
    }

    files := []string{
        usersFile, entriesFile, transactionsFile, winnersFile,
        statesFile, statsFile, adminActionsFile,
    }

    for _, file := range files {
        if err := ds.initializeFile(ctx, file); err != nil {
            return nil, err
        }
    }

    return ds, nil
}

func (ds *DriveStorage) initializeFile(ctx context.Context, filename string) error {
    fileID, err := ds.findFile(ctx, filename)
    if err != nil {
        return err
    }

    if fileID == "" {
        file := &drive.File{
            Name:     filename,
            Parents:  []string{ds.folderID},
            MimeType: "application/json",
        }

        created, err := ds.service.Files.Create(file).Context(ctx).Do()
        if err != nil {
            return fmt.Errorf("failed to create file %s: %v", filename, err)
        }
        fileID = created.Id

        emptyData := "[]"
        _, err = ds.service.Files.Update(fileID, nil).Media(strings.NewReader(emptyData)).Context(ctx).Do()
        if err != nil {
            return fmt.Errorf("failed to initialize file %s: %v", filename, err)
        }
    }

    ds.fileCache[filename] = fileID
    return nil
}

func (ds *DriveStorage) findFile(ctx context.Context, filename string) (string, error) {
    query := fmt.Sprintf("name='%s' and '%s' in parents and trashed=false", filename, ds.folderID)
    files, err := ds.service.Files.List().Q(query).Context(ctx).Do()
    if err != nil {
        return "", fmt.Errorf("failed to search for file %s: %v", filename, err)
    }

    if len(files.Files) > 0 {
        return files.Files[0].Id, nil
    }
    return "", nil
}

func (ds *DriveStorage) readFile(ctx context.Context, filename string, v interface{}) error {
    fileID, ok := ds.fileCache[filename]
    if !ok {
        return fmt.Errorf("file %s not initialized", filename)
    }

    resp, err := ds.service.Files.Get(fileID).Context(ctx).Download()
    if err != nil {
        return fmt.Errorf("failed to download file %s: %v", filename, err)
    }
    defer resp.Body.Close()

    if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
        return fmt.Errorf("failed to decode JSON from file %s: %v", filename, err)
    }

    return nil
}

func (ds *DriveStorage) writeFile(ctx context.Context, filename string, v interface{}) error {
    fileID, ok := ds.fileCache[filename]
    if !ok {
        return fmt.Errorf("file %s not initialized", filename)
    }

    data, err := json.Marshal(v)
    if err != nil {
        return fmt.Errorf("failed to marshal data: %v", err)
    }

    _, err = ds.service.Files.Update(fileID, nil).Media(strings.NewReader(string(data))).Context(ctx).Do()
    if err != nil {
        return fmt.Errorf("failed to write file %s: %v", filename, err)
    }

    return nil
}

func (ds *DriveStorage) SaveUser(ctx context.Context, user *models.User) error {
    ds.mutex.Lock()
    defer ds.mutex.Unlock()

    var users []*models.User
    if err := ds.readFile(ctx, usersFile, &users); err != nil {
        return storage.NewStorageError("SaveUser", err)
    }

    found := false
    for i, u := range users {
        if u.UserID == user.UserID {
            users[i] = user
            found = true
            break
        }
    }
    if !found {
        users = append(users, user)
    }

    return ds.writeFile(ctx, usersFile, users)
}

func (ds *DriveStorage) GetUser(ctx context.Context, userID int64) (*models.User, error) {
    ds.mutex.RLock()
    defer ds.mutex.RUnlock()

    var users []*models.User
    if err := ds.readFile(ctx, usersFile, &users); err != nil {
        return nil, storage.NewStorageError("GetUser", err)
    }

    for _, user := range users {
        if user.UserID == userID {
            return user, nil
        }
    }

    return nil, storage.NewStorageError("GetUser", fmt.Errorf("user not found"))
}

func (ds *DriveStorage) GetAllUsers(ctx context.Context, fromDate, toDate time.Time) ([]*models.User, error) {
    ds.mutex.RLock()
    defer ds.mutex.RUnlock()

    var users []*models.User
    if err := ds.readFile(ctx, usersFile, &users); err != nil {
        return nil, storage.NewStorageError("GetAllUsers", err)
    }

    if fromDate.IsZero() && toDate.IsZero() {
        return users, nil
    }

    var filtered []*models.User
    for _, user := range users {
        if (fromDate.IsZero() || !user.JoinedDate.Before(fromDate)) &&
           (toDate.IsZero() || !user.JoinedDate.After(toDate)) {
            filtered = append(filtered, user)
        }
    }

    return filtered, nil
}

func (ds *DriveStorage) SaveTransaction(ctx context.Context, txn *models.Transaction) error {
    ds.mutex.Lock()
    defer ds.mutex.Unlock()

    var transactions []*models.Transaction
    if err := ds.readFile(ctx, transactionsFile, &transactions); err != nil {
        return storage.NewStorageError("SaveTransaction", err)
    }

    transactions = append(transactions, txn)
    return ds.writeFile(ctx, transactionsFile, transactions)
}

func (ds *DriveStorage) GetTransaction(ctx context.Context, txnID string) (*models.Transaction, error) {
    ds.mutex.RLock()
    defer ds.mutex.RUnlock()

    var transactions []*models.Transaction
    if err := ds.readFile(ctx, transactionsFile, &transactions); err != nil {
        return nil, storage.NewStorageError("GetTransaction", err)
    }

    for _, txn := range transactions {
        if txn.TransactionID == txnID {
            return txn, nil
        }
    }

    return nil, storage.NewStorageError("GetTransaction", fmt.Errorf("transaction not found"))
}

func (ds *DriveStorage) GetTransactionsByDate(ctx context.Context, date time.Time) ([]*models.Transaction, error) {
    ds.mutex.RLock()
    defer ds.mutex.RUnlock()

    var transactions []*models.Transaction
    if err := ds.readFile(ctx, transactionsFile, &transactions); err != nil {
        return nil, storage.NewStorageError("GetTransactionsByDate", err)
    }

    startDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
    endDate := startDate.Add(24 * time.Hour)

    var filtered []*models.Transaction
    for _, txn := range transactions {
        if txn.Date.After(startDate) && txn.Date.Before(endDate) {
            filtered = append(filtered, txn)
        }
    }

    return filtered, nil
}

func (ds *DriveStorage) IsTransactionUsed(ctx context.Context, txnID string) (bool, error) {
    ds.mutex.RLock()
    defer ds.mutex.RUnlock()

    var transactions []*models.Transaction
    if err := ds.readFile(ctx, transactionsFile, &transactions); err != nil {
        return false, storage.NewStorageError("IsTransactionUsed", err)
    }

    for _, txn := range transactions {
        if txn.TransactionID == txnID {
            return true, nil
        }
    }

    return false, nil
}

func (ds *DriveStorage) SaveLotteryEntry(ctx context.Context, entry *models.LotteryEntry) error {
    ds.mutex.Lock()
    defer ds.mutex.Unlock()

    var entries []*models.LotteryEntry
    if err := ds.readFile(ctx, entriesFile, &entries); err != nil {
        return storage.NewStorageError("SaveLotteryEntry", err)
    }

    entries = append(entries, entry)
    return ds.writeFile(ctx, entriesFile, entries)
}

func (ds *DriveStorage) GetEntriesByDate(ctx context.Context, date time.Time) ([]*models.LotteryEntry, error) {
    ds.mutex.RLock()
    defer ds.mutex.RUnlock()

    var entries []*models.LotteryEntry
    if err := ds.readFile(ctx, entriesFile, &entries); err != nil {
        return nil, storage.NewStorageError("GetEntriesByDate", err)
    }

    startDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
    endDate := startDate.Add(24 * time.Hour)

    var filtered []*models.LotteryEntry
    for _, entry := range entries {
        if entry.EntryDate.After(startDate) && entry.EntryDate.Before(endDate) {
            filtered = append(filtered, entry)
        }
    }

    return filtered, nil
}

func (ds *DriveStorage) SaveUserState(ctx context.Context, state *models.UserState) error {
    ds.mutex.Lock()
    defer ds.mutex.Unlock()

    var states []*models.UserState
    if err := ds.readFile(ctx, statesFile, &states); err != nil {
        return storage.NewStorageError("SaveUserState", err)
    }

    found := false
    for i, s := range states {
        if s.UserID == state.UserID {
            states[i] = state
            found = true
            break
        }
    }

    if !found {
        states = append(states, state)
    }

    return ds.writeFile(ctx, statesFile, states)
}

func (ds *DriveStorage) GetUserState(ctx context.Context, userID int64) (*models.UserState, error) {
    ds.mutex.RLock()
    defer ds.mutex.RUnlock()

    var states []*models.UserState
    if err := ds.readFile(ctx, statesFile, &states); err != nil {
        return nil, storage.NewStorageError("GetUserState", err)
    }

    for _, state := range states {
        if state.UserID == userID {
            return state, nil
        }
    }

    // Return new state if not found
    return &models.UserState{
        UserID:          userID,
        CurrentState:    "",
        InvalidAttempts: 0,
        LastUpdated:     time.Now(),
    }, nil
}

func (ds *DriveStorage) DeleteUserState(ctx context.Context, userID int64) error {
    ds.mutex.Lock()
    defer ds.mutex.Unlock()

    var states []*models.UserState
    if err := ds.readFile(ctx, statesFile, &states); err != nil {
        return storage.NewStorageError("DeleteUserState", err)
    }

    for i, state := range states {
        if state.UserID == userID {
            states = append(states[:i], states[i+1:]...)
            break
        }
    }

    return ds.writeFile(ctx, statesFile, states)
}
