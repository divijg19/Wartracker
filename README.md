# War Tracker Discord Bot (Go)

Dedicated in part to Yargit, project originator.

A Discord slash-command bot for a mobile gaming guild (<=25 members). Tracks member registration, war orders, lumber, and availability windows.

## Features
- /register – register or update in-game name
- /order – set war orders
- /lumber – set lumber
- /availability – interactive select menu for time slot
- /roster add/remove – manage roster
- /list availability – show availability list (embed)
- /list current – show resources list (embed)
- /syncroles [role-id] – resync stored roles from guild members (optionally restricted to a role)

## Tech Stack
- Go + discordgo
- SQLite (database/sql + mattn/go-sqlite3)

## Configuration
Create `config.json` in project root. Legacy single-guild config is supported, but multi-guild is recommended.

Example (multi-guild):
```json
{
  "BotToken": "YOUR_DISCORD_BOT_TOKEN_HERE",
  "DBPath": "data/guild_data.db",
  "Guilds": [
    {
      "GuildID": "YOUR_GUILD_ID",
      "LeaderRoleIDs": ["ROLE_ID_A", "ROLE_ID_B"],
      "LeaderUserIDs": ["USER_ID_OPTIONAL"]
    }
  ],
  "TLSInsecureSkipVerify": false,
  "CustomRootCAPath": ""
}
```

Notes:
- `LeaderRoleIDs` doubles as a “tracked roles” set; when a user registers, the bot snapshots the first matching role they already have and stores it in the DB for segmentation.
- You can still use legacy fields (`GuildID`, `LeaderRoleID`) if preferred.

## Run
```
go mod tidy
go run ./cmd/bot
```

## Database
Auto-creates the database (default `guild_data.db` or `DBPath`) with tables `members` and `leader`.

Zero-downtime: the bot uses a SQLite-backed leader lease to support blue/green deploys. Start the new instance first (it waits as standby), then stop the old one to cut over instantly.

TLS in corp/proxy environments: set `CustomRootCAPath` to your CA PEM, or temporarily set `TLSInsecureSkipVerify` to true for dev only.

## Role Sync
- Automatic: Runs once at startup, then monthly, syncing members and snapshotting a relevant role.
- Manual: Use `/syncroles` to sync now. Optionally pass `role-id` to limit to a specific role.

## Graceful Shutdown
CTRL+C triggers session close and DB close.

## Notes
Commands register per guild for fast iteration. Promote to global by changing registration logic (not included here).
