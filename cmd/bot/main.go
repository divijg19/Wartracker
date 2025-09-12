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
	// Load configuration
	cfg, err := bot.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Basic validation to avoid Discord 4004 auth failures with placeholders
	if cfg.BotToken == "" || cfg.BotToken == "YOUR_DISCORD_BOT_TOKEN_HERE" {
		log.Fatal("config error: BotToken is missing or placeholder; update config.json with a real token")
	}
	if (cfg.GuildID == "" || cfg.GuildID == "YOUR_TESTING_SERVER_ID_HERE") && len(cfg.GuildList()) == 0 {
		log.Fatal("config error: No guilds configured; set GuildID or provide Guilds[] in config.json")
	}

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
