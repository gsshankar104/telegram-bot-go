# Telegram Bot GitHub Repository Setup Guide

## Repository Name Suggestion
Name your repository: `telegram-bot-go`

## Steps to Push Code to GitHub

1. Create a new repository on GitHub
   - Go to github.com
   - Click "New repository"
   - Name it `telegram-bot-go`
   - Keep it public or private as per your preference
   - Don't initialize with README (we'll push our existing one)

2. Initialize Git in your local project (if not already done)
```bash
cd telegram-bot
git init
```

3. Add your files to Git
```bash
git add .
```

4. Create initial commit
```bash
git commit -m "Initial commit: Telegram bot implementation"
```

5. Link your local repository to GitHub
```bash
git remote add origin https://github.com/YOUR_USERNAME/telegram-bot-go.git
```

6. Push your code
```bash
git push -u origin main
```

## GitHub Actions Workflow
The repository includes a GitHub Actions workflow in `.github/workflows/go.yml` that will automatically build and run your bot.

## Important Notes
- Make sure to set up your bot token as a GitHub Secret named `TELEGRAM_BOT_TOKEN`
- Update the repository URL in go.mod to match your GitHub username