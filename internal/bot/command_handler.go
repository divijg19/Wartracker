package bot

import (
	"context"
	"log"

	"github.com/bwmarrin/discordgo"
)

// RegisterHandlers wires the interaction and component handlers.
func RegisterHandlers(b *Bot) {
	b.Session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type == discordgo.InteractionApplicationCommand {
			handleSlashCommand(b, s, i)
		} else if i.Type == discordgo.InteractionMessageComponent {
			handleComponentInteraction(b, s, i)
		}
	})
}

// registerSlashCommands defines and registers commands on the guild.
func (b *Bot) registerSlashCommands() error {
	guildID := b.Config.GuildID
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "register",
			Description: "Register or update your in-game name",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "in-game-name",
					Description: "Your in-game name",
					Required:    true,
				},
			},
		},
		{
			Name:        "order",
			Description: "Set your current War Orders count",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "amount",
					Description: "Number of War Orders",
					Required:    true,
				},
			},
		},
		{
			Name:        "lumber",
			Description: "Set your current Lumber count",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "amount",
					Description: "Amount of Lumber",
					Required:    true,
				},
			},
		},
		{
			Name:        "availability",
			Description: "Set your availability time slot",
		},
		{
			Name:        "help",
			Description: "Show bot commands and usage",
		},
		{
			Name:        "tutorial",
			Description: "Show a short getting-started tutorial",
		},
		// Roster subcommands
		{
			Name:        "roster",
			Description: "Leader: manage roster",
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
		// Listing commands
		{
			Name:        "list",
			Description: "Leader: list guild data",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "availability",
					Description: "Show availability list",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "current",
					Description: "Show current orders and lumber",
				},
			},
		},
	}

	for _, cmd := range commands {
		_, err := b.Session.ApplicationCommandCreate(b.Session.State.User.ID, guildID, cmd)
		if err != nil {
			return err
		}
	}
	log.Printf("Registered %d commands in guild %s", len(commands), guildID)
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
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Content: content, Flags: discordgo.MessageFlagsEphemeral},
		})
	}
}
