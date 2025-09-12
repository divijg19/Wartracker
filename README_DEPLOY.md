Quick deploy checklist

Docker (recommended):
- Ensure Docker Engine and Docker Compose v2 are installed.
- Create a `.env` with BOT_TOKEN and optional DB_PATH if you want a specific path inside the container.
- Run `docker compose up -d --build` from the repo root.
- Confirm the container is running: `docker ps` and `docker logs -f <container>`.

Backups:
- Use `deploy/backup_db.sh` to take point-in-time backups. Mount the data volume to a host path if you want to run backups on the host.

Native binary (if you prefer no Docker):
- Build: `CGO_ENABLED=1 go build -o wartracker ./cmd/bot`
- Create a `service` user and systemd unit similar to `deploy/systemd-run-binary.service`.
- Place `guild_data.db` in a persistent directory and set `DB_PATH`.

Security:
- Never commit `config.json` with a real token. Use environment variables or a secrets store.
- Consider storing backups off-host (S3, remote share) and rotate them.
