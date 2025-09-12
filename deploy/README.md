Wartracker deployment
=====================

This folder contains example artifacts to run the bot 24/7 with Docker Compose or as a systemd-managed Docker Compose service, and to keep periodic backups of the SQLite DB.

Docker (recommended)
--------------------

1. Create an env file with your bot token next to the compose file:

```env
BOT_TOKEN=your_bot_token_here
```

2. Build and start in detached mode:

```bash
docker compose -f docker-compose.prod.yml up -d --build
```

3. Check logs:

```bash
docker compose -f docker-compose.prod.yml logs -f wartracker
```

Backups
-------

- Backups are stored in the `backups` volume. The `backup` sidecar performs a daily backup using sqlite3 .backup when available.
- To list backups:

```bash
docker run --rm -v wartracker_backups:/backups alpine ls -la /backups
```

Restore
-------

1. Stop the bot:

```bash
docker compose -f docker-compose.prod.yml stop wartracker
```

2. Copy a backup to the data volume (example picks the latest file):

```bash
LATEST=$(docker run --rm -v wartracker_backups:/backups alpine sh -c "ls -1t /backups | head -n1")
docker run --rm -v wartracker_backups:/backups -v wartracker_data:/data alpine sh -c "cp /backups/$LATEST /data/guild_data.db"
```

3. Start the bot again:

```bash
docker compose -f docker-compose.prod.yml up -d wartracker
```

Systemd (alternative)
----------------------

1. Copy the repository to `/opt/wartracker` on the host.
2. Place `docker-compose.prod.yml` in `/opt/wartracker`.
3. Install the unit file to `/etc/systemd/system/wartracker.service` and run:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now wartracker.service
sudo journalctl -u wartracker -f
```

Notes
-----

- The SQLite DB is stored on the named volume `data`. You can instead bind-mount a host directory in `docker-compose.prod.yml`.
- If you expect high write throughput, ensure the host disk is reliable — consider scheduling more frequent backups.
Deployment notes

Run via Docker (recommended):

1. Build and start with docker-compose (set BOT_TOKEN env):

   export BOT_TOKEN=your_bot_token_here
   docker compose up -d --build

2. Data is stored in the named volume `data`. Backups can be taken from the host by running the included script in `deploy/backup_db.sh` (mount the data volume or copy out the DB file).

Run as systemd service (Linux):

- Build the Docker image and use a systemd unit that runs `docker compose up` or run the binary directly using the system unit example in this repo.

Windows service (optional):

- Use NSSM to wrap the binary and point the working dir to the repo folder and set the env var BOT_TOKEN and DB_PATH.

Notes:
- The container sets DB_PATH to /data/guild_data.db; you can override it by passing DB_PATH env var.
- Ensure the bot has correct invite scopes (bot + applications.commands) and necessary permissions in each guild.
