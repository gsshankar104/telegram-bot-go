# Telegram Lottery Bot

A Telegram bot for managing lottery entries with Google Drive storage backend.

## Features

- Multiple ticket price options
- Transaction tracking
- Lucky number selection
- Admin controls
- Winner selection system
- Google Drive storage backend
- Rate limiting
- User state management

## Prerequisites

- Go 1.19 or higher
- A Telegram Bot Token (from [@BotFather](https://t.me/BotFather))
- Google Cloud Project with Drive API enabled
- Google Drive API credentials

## Setup

1. Clone the repository:
```bash
git clone https://github.com/yourusername/telegram-bot.git
cd telegram-bot
```

2. Configure Google Drive API:
   - Go to [Google Cloud Console](https://console.cloud.google.com)
   - Create a new project or select existing one
   - Enable Drive API
   - Create credentials (OAuth 2.0 Client ID)
   - Download credentials and save as `credentials.json` in project root

3. Create a Drive folder for storage:
   - Create a folder in Google Drive
   - Copy the folder ID from the URL
   - Update `config/config.yaml` with the folder ID

4. Update configuration:
   - Copy `config/config.yaml` and modify with your settings:
     ```yaml
     bot:
       token: "YOUR_BOT_TOKEN"    # From @BotFather
       name: "Lottery Bot"        # Your bot's name

     admin:
       ids:                       # Admin Telegram User IDs
         - "YOUR_TELEGRAM_ID"

     database:
       drive_folder_id: "YOUR_GOOGLE_DRIVE_FOLDER_ID"

     channels:
       lottery_proof: "https://t.me/YOUR_LOTTERY_PROOF_CHANNEL"
       lottery_win: "https://t.me/YOUR_LOTTERY_WIN_CHANNEL"

     payment:
       qr_code_link: "YOUR_QR_CODE_IMAGE_URL"

     tickets:
       prices:                    # Available ticket prices
         - 100
         - 200
         - 500
         - 1000

     limits:
       max_invalid_attempts: 3    # Max invalid inputs before reset
       command_rate_limit: 5      # Commands per minute
     ```

5. Build the bot:
```bash
make build
```

## Running

```bash
./bin/bot
```

The bot will start and listen for updates. Press Ctrl+C to stop gracefully.

## Commands

### User Commands
- `/start` - Start the bot and view ticket options
- `/view_data` - View your lottery entries

### Admin Commands
- `/view_data user @username` - View user data
- `/view_data txn <transaction_id>` - View transaction data
- `/view_data date YYYY-MM-DD` - View lottery entries for date
- `/view_data txndate YYYY-MM-DD` - View transactions for date
- `/select_winner` - Start winner selection process
- `/export_all_users` - Export users data to CSV

## Development

### Project Structure
```
telegram-bot/
├── cmd/
│   └── bot/
│       └── main.go
├── internal/
│   ├── bot/
│   │   └── bot.go
│   ├── config/
│   │   └── config.go
│   ├── models/
│   │   └── models.go
│   └── storage/
│       ├── storage.go
│       └── drive/
│           └── drive.go
├── config/
│   └── config.yaml
├── credentials.json
├── Makefile
└── README.md
```

### Building
```bash
make build    # Build the bot
make clean    # Clean build artifacts
```

### Testing
```bash
make test     # Run tests
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support, contact the administrators through the Lottery Win channel specified in your config.