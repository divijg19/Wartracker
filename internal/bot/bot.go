package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/divijg19/Wartracker/internal/storage"
)

// Config structure for config.json
// Keep sensitive values outside the source.
type Config struct {
	BotToken     string `json:"BotToken"`
	GuildID      string `json:"GuildID"`
	LeaderRoleID string `json:"LeaderRoleID"`
}

// Bot holds session, config, and database handle.
type Bot struct {
	Session *discordgo.Session
	Config  *Config
	DB      *storage.DB
}

// LoadConfig reads a JSON config file into Config struct.
func LoadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil { return nil, err }
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil { return nil, err }
	return &cfg, nil
}

// New creates a new Bot and wires handlers (but does not open the session).
func New(cfg *Config, db *storage.DB) (*Bot, error) {
	s, err := discordgo.New("Bot "+cfg.BotToken)
	if err != nil { return nil, err }
	b := &Bot{Session: s, Config: cfg, DB: db}
	RegisterHandlers(b)
	return b, nil
}

// Open starts the session and registers commands.
func (b *Bot) Open(ctx context.Context) error {
	if err := b.Session.Open(); err != nil { return err }
	if err := b.registerSlashCommands(); err != nil { return fmt.Errorf("register commands: %w", err) }
	return nil
}

// Close shuts everything down gracefully.
func (b *Bot) Close() {
	_ = b.Session.Close()
	_ = b.DB.Close()
}

// WaitForInterrupt blocks until an OS interrupt signal is received.
func (b *Bot) WaitForInterrupt() {
	fmt.Println("Bot running. Press CTRL+C to exit.")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	fmt.Println("Termination signal received. Closing...")
	time.Sleep(500 * time.Millisecond)
}
