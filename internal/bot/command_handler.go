package bot

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

// RegisterHandlers wires the interaction and component handlers.
func RegisterHandlers(b *Bot) {
	b.Session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		// Use a tagged switch on the interaction type (staticcheck QF1003)
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			handleSlashCommand(b, s, i)
		case discordgo.InteractionMessageComponent:
			handleComponentInteraction(b, s, i)
		}
	})
}

// registerSlashCommands defines and registers commands for each configured guild.
func (b *Bot) registerSlashCommands() error {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "register",
			Description: "Register or update your in-game name",
			Options: []*discordgo.ApplicationCommandOption{{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "in-game-name",
				Description: "Your in-game name",
				Required:    true,
			}},
		},
		{
			Name:        "order",
			Description: "Set your current War Orders count",
			Options: []*discordgo.ApplicationCommandOption{{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "amount",
				Description: "Number of War Orders",
				Required:    true,
			}},
		},
		{
			Name:        "lumber",
			Description: "Set your current Lumber count",
			Options: []*discordgo.ApplicationCommandOption{{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "amount",
				Description: "Amount of Lumber",
				Required:    true,
			}},
		},
		{Name: "availability", Description: "Set your availability time slot"},
		{Name: "help", Description: "Show bot commands and usage"},
		{Name: "tutorial", Description: "Show a short getting-started tutorial"},
		{
			Name:        "roster",
			Description: "Manage roster",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add",
					Description: "Add or update a member",
					Options: []*discordgo.ApplicationCommandOption{
						{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "Discord user", Required: true},
						{Type: discordgo.ApplicationCommandOptionString, Name: "in-game-name", Description: "In-game name", Required: true},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "remove",
					Description: "Remove a member",
					Options: []*discordgo.ApplicationCommandOption{
						{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "Discord user", Required: true},
					},
				},
			},
		},
		{
			Name:        "list",
			Description: "List guild data",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "availability", Description: "Show availability list"},
				{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "current", Description: "Show current orders and lumber"},
			},
		},
		{
			Name:        "syncroles",
			Description: "Sync stored roles from guild members (optional: filter by role id)",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "role-id", Description: "Only sync members who have this role", Required: false},
			},
		},
	}

	guilds := b.Config.GuildList()
	// If no guilds configured, register global commands so the bot works after invite
	appID := b.Session.State.User.ID
	anyGuildOK := false
	if len(guilds) == 0 {
		if _, err := b.Session.ApplicationCommandBulkOverwrite(appID, "", commands); err != nil {
			return fmt.Errorf("register global commands: %w", err)
		}
		log.Printf("Registered %d global commands (no guilds configured)", len(commands))
		return nil
	}
	for _, g := range guilds {
		if g.GuildID == "" {
			continue
		}
		if _, err := b.Session.ApplicationCommandBulkOverwrite(appID, g.GuildID, commands); err != nil {
			// Hint if missing access
			if re, ok := err.(*discordgo.RESTError); ok && re.Response != nil && re.Response.StatusCode == 403 {
				invite := fmt.Sprintf("https://discord.com/api/oauth2/authorize?client_id=%s&scope=bot%%20applications.commands", appID)
				log.Printf("WARN: Missing Access to guild %s for command registration. Ensure the bot is invited to that server with 'bot' and 'applications.commands' scopes. Invite URL: %s", g.GuildID, invite)
				continue
			}
			log.Printf("ERROR: register commands for guild %s: %v", g.GuildID, err)
			continue
		}
		anyGuildOK = true
		log.Printf("Registered %d commands in guild %s", len(commands), g.GuildID)
	}
	if anyGuildOK {
		return nil
	}
	// Fallback: register global commands so at least slash commands exist when the bot is invited correctly.
	if _, err := b.Session.ApplicationCommandBulkOverwrite(appID, "", commands); err != nil {
		return fmt.Errorf("register global commands: %w", err)
	}
	log.Printf("Registered %d global commands (fallback)", len(commands))
	return nil
}

// handleSlashCommand routes slash command invocations.
func handleSlashCommand(b *Bot, s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	switch i.ApplicationCommandData().Name {
	case "register":
		handleRegister(b, s, i, ctx)
	case "order":
		handleOrder(b, s, i, ctx)
	case "lumber":
		handleLumber(b, s, i, ctx)
	case "availability":
		handleAvailability(b, s, i, ctx)
	case "help":
		handleHelp(b, s, i, ctx)
	case "tutorial":
		handleTutorial(b, s, i, ctx)
	case "roster":
		handleRoster(b, s, i, ctx)
	case "list":
		handleList(b, s, i, ctx)
	case "syncroles":
		handleSyncRoles(b, s, i, ctx)
	}
}

// handleComponentInteraction processes select menu submissions for availability.
func handleComponentInteraction(b *Bot, s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.MessageComponentData()
	if data.CustomID == availabilitySelectID {
		sel := "Not Set"
		if len(data.Values) > 0 {
			sel = data.Values[0]
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		err := b.DB.UpdateAvailability(ctx, i.Member.User.ID, sel)
		var content string
		if err != nil {
			content = "Failed to set availability: " + err.Error()
		} else {
			content = "Your availability has been set to " + sel
		}
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: content, Flags: discordgo.MessageFlagsEphemeral},
		})
	}
}
