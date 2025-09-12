package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/divijg19/Wartracker/internal/bot"
	"github.com/divijg19/Wartracker/internal/storage"
)

func main() {
	// Load configuration
	cfg, err := bot.LoadConfig("config.json")
	if err != nil { log.Fatalf("load config: %v", err) }

	// Init database (guild_data.db in working directory)
	db, err := storage.NewConnection("guild_data.db")
	if err != nil { log.Fatalf("db: %v", err) }

	// Create bot
	b, err := bot.New(cfg, db)
	if err != nil { log.Fatalf("bot: %v", err) }
	defer b.Close()

	// Open session and register commands
	if err := b.Open(context.Background()); err != nil { log.Fatalf("open: %v", err) }

	// Graceful shutdown
	b.WaitForInterrupt()
	fmt.Println("Shutdown complete.")
	os.Exit(0)
}
