package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/divijg19/Wartracker/internal/bot"
	"github.com/divijg19/Wartracker/internal/storage"
)

func main() {
	// Load configuration from file if present, then override with environment variables for smooth container deploys
	var cfg *bot.Config
	if _, statErr := os.Stat("config.json"); statErr == nil {
		c, err := bot.LoadConfig("config.json")
		if err != nil {
			log.Fatalf("load config: %v", err)
		}
		cfg = c
	} else {
		cfg = &bot.Config{}
	}

	// Environment overrides
	if v := os.Getenv("BOT_TOKEN"); v != "" {
		cfg.BotToken = v
	}
	if v := os.Getenv("DB_PATH"); v != "" {
		cfg.DBPath = v
	}
	// Guilds: support single GUILD_ID or comma-separated GUILD_IDS
	if v := os.Getenv("GUILD_ID"); v != "" {
		cfg.Guilds = []bot.GuildConfig{{GuildID: v}}
	} else if v := os.Getenv("GUILD_IDS"); v != "" {
		// split by comma
		ids := []string{}
		for _, part := range splitAndTrim(v, ',') {
			if part != "" {
				ids = append(ids, part)
			}
		}
		if len(ids) > 0 {
			cfg.Guilds = make([]bot.GuildConfig, 0, len(ids))
			for _, id := range ids {
				cfg.Guilds = append(cfg.Guilds, bot.GuildConfig{GuildID: id})
			}
		}
	}
	if v := os.Getenv("TLS_INSECURE_SKIP_VERIFY"); v != "" {
		if v == "1" || v == "true" || v == "TRUE" {
			cfg.TLSInsecureSkipVerify = true
		}
	}
	if v := os.Getenv("CUSTOM_ROOT_CA_PATH"); v != "" {
		cfg.CustomRootCAPath = v
	}

	// Basic validation to avoid Discord 4004 auth failures with placeholders
	if cfg.BotToken == "" || cfg.BotToken == "YOUR_DISCORD_BOT_TOKEN_HERE" {
		log.Fatal("config error: BotToken is missing or placeholder; update config.json with a real token")
	}
	// No hard-fail if no guilds configured; we'll register global commands as a fallback

	// Resolve DB path
	dbPath := cfg.DBPath
	if dbPath == "" {
		dbPath = "guild_data.db"
	}
	if dir := filepath.Dir(dbPath); dir != "." && dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}

	// Init database
	db, err := storage.NewConnection(dbPath)
	if err != nil {
		log.Fatalf("db: %v", err)
	}

	// Create bot
	b, err := bot.New(cfg, db)
	if err != nil {
		log.Fatalf("bot: %v", err)
	}
	defer b.Close()

	// Leader election (active/standby) for zero-downtime deploys
	host, _ := os.Hostname()
	instanceID := fmt.Sprintf("%s-%d", host, os.Getpid())
	lease := 10 * time.Second
	baseCtx := context.Background()

	// Acquire leadership before connecting to Discord
	for {
		cctx, cancel := storage.WithTimeout(baseCtx)
		ok, err := db.TryAcquireLeader(cctx, instanceID, lease, true)
		cancel()
		if err != nil {
			log.Printf("leader acquire error: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}
		if ok {
			log.Printf("leadership acquired by %s", instanceID)
			break
		}
		time.Sleep(2 * time.Second)
	}

	// Renew the lease periodically
	stopRenew := make(chan struct{})
	go func() {
		t := time.NewTicker(lease / 2)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				cctx, cancel := storage.WithTimeout(baseCtx)
				if err := db.RenewLeader(cctx, instanceID); err != nil {
					log.Printf("leader renew error: %v", err)
				}
				cancel()
			case <-stopRenew:
				return
			}
		}
	}()

	// Open session and register commands
	if err := b.Open(context.Background()); err != nil {
		log.Fatalf("open: %v", err)
	}

	// Graceful shutdown
	b.WaitForInterrupt()
	// Release leader
	close(stopRenew)
	cctx, cancel := storage.WithTimeout(baseCtx)
	_ = db.ReleaseLeader(cctx, instanceID)
	cancel()

	fmt.Println("Shutdown complete.")
	os.Exit(0)
}

// splitAndTrim splits a string by sep and trims whitespace around parts.
func splitAndTrim(s string, sep rune) []string {
	out := make([]string, 0)
	cur := make([]rune, 0, len(s))
	for _, r := range s {
		if r == sep {
			part := string(cur)
			out = append(out, trimSpace(part))
			cur = cur[:0]
		} else {
			cur = append(cur, r)
		}
	}
	out = append(out, trimSpace(string(cur)))
	return out
}

func trimSpace(s string) string {
	i := 0
	j := len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t' || s[j-1] == '\n' || s[j-1] == '\r') {
		j--
	}
	return s[i:j]
}
