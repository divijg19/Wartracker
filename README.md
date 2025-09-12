# War Tracker Discord Bot (Go)

A Discord slash-command bot for a mobile gaming guild (<=25 members). Tracks member registration, war orders, lumber, and availability windows.

## Features
- /register – register or update in-game name
- /order – set war orders
- /lumber – set lumber
- /availability – interactive select menu for time slot
- /roster add/remove – leadership only
- /list availability – show availability list (embed)
- /list current – show resources list (embed)

## Tech Stack
- Go + discordgo
- SQLite (database/sql + mattn/go-sqlite3)

## Configuration
Create `config.json` in project root:
```json
{
  "BotToken": "YOUR_DISCORD_BOT_TOKEN_HERE",
  "GuildID": "YOUR_TESTING_SERVER_ID_HERE",
  "LeaderRoleID": "THE_DISCORD_ROLE_ID_FOR_LEADERSHIP"
}
```

## Run
```
go mod tidy
go run ./cmd/bot
```

## Database
Auto-creates `guild_data.db` with table `members`.

## Graceful Shutdown
CTRL+C triggers session close and DB close.

## Notes
Commands register per guild for fast iteration. Promote to global by changing registration logic (not included here).
