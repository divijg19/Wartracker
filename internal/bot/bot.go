package bot

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
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
	BotToken string `json:"BotToken"`
	// Deprecated single-guild settings (backward compatibility)
	GuildID      string `json:"GuildID,omitempty"`
	LeaderRoleID string `json:"LeaderRoleID,omitempty"`
	// New multi-guild settings
	Guilds []GuildConfig `json:"Guilds,omitempty"`
	DBPath string        `json:"DBPath,omitempty"`
	// TLS options for environments with intercepting proxies or custom CAs
	TLSInsecureSkipVerify bool   `json:"TLSInsecureSkipVerify,omitempty"`
	CustomRootCAPath      string `json:"CustomRootCAPath,omitempty"`
}

// GuildConfig contains per-guild leadership settings.
type GuildConfig struct {
	GuildID       string   `json:"GuildID"`
	LeaderRoleIDs []string `json:"LeaderRoleIDs,omitempty"`
	LeaderUserIDs []string `json:"LeaderUserIDs,omitempty"`
}

// GuildList returns configured guilds, falling back to deprecated fields.
func (c *Config) GuildList() []GuildConfig {
	if len(c.Guilds) > 0 {
		return c.Guilds
	}
	if c.GuildID != "" {
		gc := GuildConfig{GuildID: c.GuildID}
		if c.LeaderRoleID != "" {
			gc.LeaderRoleIDs = []string{c.LeaderRoleID}
		}
		return []GuildConfig{gc}
	}
	return nil
}

// GuildConfigFor returns the per-guild config entry, falling back to legacy fields.
func (c *Config) GuildConfigFor(guildID string) *GuildConfig {
	if guildID == "" {
		return nil
	}
	for _, g := range c.Guilds {
		if g.GuildID == guildID {
			gg := g
			return &gg
		}
	}
	if c.GuildID == guildID {
		gc := GuildConfig{GuildID: c.GuildID}
		if c.LeaderRoleID != "" {
			gc.LeaderRoleIDs = []string{c.LeaderRoleID}
		}
		return &gc
	}
	return nil
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
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// New creates a new Bot and wires handlers (but does not open the session).
func New(cfg *Config, db *storage.DB) (*Bot, error) {
	s, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		return nil, err
	}
	// Minimal required intents. Members intent is privileged; allow disabling via env.
	intents := discordgo.IntentsGuilds
	if v := os.Getenv("ENABLE_GUILD_MEMBERS_INTENT"); v == "" || v == "1" || v == "true" || v == "TRUE" {
		intents |= discordgo.IntentsGuildMembers
	}
	s.Identify.Intents = intents
	// Configure HTTP client TLS if requested
	if cfg.TLSInsecureSkipVerify || cfg.CustomRootCAPath != "" {
		tlsCfg := &tls.Config{InsecureSkipVerify: cfg.TLSInsecureSkipVerify}
		if cfg.CustomRootCAPath != "" {
			pool, _ := x509.SystemCertPool()
			if pool == nil {
				pool = x509.NewCertPool()
			}
			if pem, err := os.ReadFile(cfg.CustomRootCAPath); err == nil {
				if ok := pool.AppendCertsFromPEM(pem); ok {
					tlsCfg.RootCAs = pool
				}
			}
		}
		s.Client = &http.Client{Transport: &http.Transport{TLSClientConfig: tlsCfg}}
	}
	b := &Bot{Session: s, Config: cfg, DB: db}
	RegisterHandlers(b)
	return b, nil
}

// Open starts the session and registers commands.
func (b *Bot) Open(ctx context.Context) error {
	if err := b.Session.Open(); err != nil {
		return err
	}
	if err := b.registerSlashCommands(); err != nil {
		return fmt.Errorf("register commands: %w", err)
	}
	// Start background monthly role resync
	go b.backgroundMonthlyRoleResync()
	return nil
}

// Close shuts everything down gracefully.
func (b *Bot) Close() {
	_ = b.Session.Close()
	_ = b.DB.Close()
}

// backgroundMonthlyRoleResync periodically updates stored roles for guild members.
func (b *Bot) backgroundMonthlyRoleResync() {
	// wait a bit after startup
	time.Sleep(10 * time.Second)
	ticker := time.NewTicker(30 * 24 * time.Hour)
	defer ticker.Stop()
	// run once on start as well
	b.resyncAllGuildRoles()
	for range ticker.C {
		b.resyncAllGuildRoles()
	}
}

func (b *Bot) resyncAllGuildRoles() {
	guilds := b.Config.GuildList()
	if len(guilds) == 0 {
		return
	}
	for _, g := range guilds {
		if g.GuildID == "" {
			continue
		}
		// paginate through guild members
		after := ""
		for {
			members, err := b.Session.GuildMembers(g.GuildID, after, 1000)
			if err != nil || len(members) == 0 {
				break
			}
			for _, m := range members {
				// Insert placeholder name if needed
				ctx, cancel := storage.WithTimeout(context.Background())
				_ = b.DB.InsertMemberIfMissing(ctx, m.User.ID, m.User.Username)
				// snapshot role
				if role := firstConfiguredRole(b, g.GuildID, m); role != "" {
					_ = b.DB.UpdateMemberRole(ctx, m.User.ID, role)
				}
				cancel()
			}
			after = members[len(members)-1].User.ID
			if len(members) < 1000 {
				break
			}
		}
	}
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
