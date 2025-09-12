package bot

import (
	"context"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/divijg19/Wartracker/internal/storage"
)

const availabilitySelectID = "availability_select_menu"

var availabilityOptions = []string{
	"16:00-18:00 GMT",
	"18:00-20:00 GMT",
	"20:00-22:00 GMT",
	"22:00-00:00 GMT",
	"Not Available",
}

// Utility: check if user has leader role.
func hasLeaderRole(leaderRoleID string, member *discordgo.Member) bool {
	for _, r := range member.Roles {
		if r == leaderRoleID {
			return true
		}
	}
	return false
}

func ephemeralErrorRespond(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: msg, Flags: discordgo.MessageFlagsEphemeral},
	})
}

func ephemeralOK(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: msg, Flags: discordgo.MessageFlagsEphemeral},
	})
}

// /register
func handleRegister(b *Bot, s *discordgo.Session, i *discordgo.InteractionCreate, ctx context.Context) {
	name := i.ApplicationCommandData().Options[0].StringValue()
	c, cancel := storage.WithTimeout(ctx)
	defer cancel()
	exists, _ := b.DB.EnsureMemberExists(c, i.Member.User.ID)
	_ = b.DB.UpsertMember(c, i.Member.User.ID, name)
	if exists {
		ephemeralOK(s, i, "Your in-game name has been updated to "+name+".")
	} else {
		ephemeralOK(s, i, "You have been registered as "+name+".")
	}
}

// /order
func handleOrder(b *Bot, s *discordgo.Session, i *discordgo.InteractionCreate, ctx context.Context) {
	amount := int(i.ApplicationCommandData().Options[0].IntValue())
	c, cancel := storage.WithTimeout(ctx)
	defer cancel()
	if err := b.DB.UpdateOrders(c, i.Member.User.ID, amount); err != nil {
		ephemeralErrorRespond(s, i, "You must /register first.")
		return
	}
	ephemeralOK(s, i, "Your War Orders have been set to "+strconv.Itoa(amount)+".")
}

// /lumber
func handleLumber(b *Bot, s *discordgo.Session, i *discordgo.InteractionCreate, ctx context.Context) {
	amount := int(i.ApplicationCommandData().Options[0].IntValue())
	c, cancel := storage.WithTimeout(ctx)
	defer cancel()
	if err := b.DB.UpdateLumber(c, i.Member.User.ID, amount); err != nil {
		ephemeralErrorRespond(s, i, "You must /register first.")
		return
	}
	ephemeralOK(s, i, "Your Lumbers have been set to "+strconv.Itoa(amount)+".")
}

// /availability shows select menu
func handleAvailability(b *Bot, s *discordgo.Session, i *discordgo.InteractionCreate, ctx context.Context) {
	// Build select menu options
	var opts []discordgo.SelectMenuOption
	for _, o := range availabilityOptions {
		opts = append(opts, discordgo.SelectMenuOption{Label: o, Value: o})
	}
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Select your availability:",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					&discordgo.SelectMenu{CustomID: availabilitySelectID, Placeholder: "Choose time slot", Options: opts},
				}},
			},
		},
	})
}

// /help shows a concise command overview
func handleHelp(b *Bot, s *discordgo.Session, i *discordgo.InteractionCreate, ctx context.Context) {
	embed := &discordgo.MessageEmbed{
		Title:       "Wartracker Bot Help",
		Description: "Slash commands overview",
		Color:       0x7289DA,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "/register in-game-name", Value: "Register or update your in-game name.", Inline: false},
			{Name: "/order amount", Value: "Set your current War Orders.", Inline: false},
			{Name: "/lumber amount", Value: "Set your current Lumber.", Inline: false},
			{Name: "/availability", Value: "Pick your 2-hour GMT window via a dropdown.", Inline: false},
			{Name: "/roster add/remove", Value: "Leaders: manage members.", Inline: false},
			{Name: "/list availability|current", Value: "Leaders: show availability or current resources.", Inline: false},
			{Name: "/tutorial", Value: "Quick start walkthrough.", Inline: false},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: "All user commands reply ephemerally."},
	}
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Flags: discordgo.MessageFlagsEphemeral},
	})
}

// /tutorial shows a short getting-started guide
func handleTutorial(b *Bot, s *discordgo.Session, i *discordgo.InteractionCreate, ctx context.Context) {
	embed := &discordgo.MessageEmbed{
		Title: "Getting Started",
		Color: 0x00CC99,
		Description: strings.Join([]string{
			"1) Use /register to set your in-game name.",
			"2) Use /order and /lumber to set your current resources.",
			"3) Use /availability to choose your usual 2-hour GMT time slot.",
			"4) Leaders can manage with /roster and share lists via /list.",
		}, "\n"),
	}
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Flags: discordgo.MessageFlagsEphemeral},
	})
}

// /roster add|remove (leader only)
func handleRoster(b *Bot, s *discordgo.Session, i *discordgo.InteractionCreate, ctx context.Context) {
	if !hasLeaderRole(b.Config.LeaderRoleID, i.Member) {
		ephemeralErrorRespond(s, i, "You do not have permission to use this command.")
		return
	}
	data := i.ApplicationCommandData()
	sub := data.Options[0]
	switch sub.Name {
	case "add":
		user := sub.Options[0].UserValue(s)
		name := sub.Options[1].StringValue()
		c, cancel := storage.WithTimeout(ctx)
		defer cancel()
		_ = b.DB.UpsertMember(c, user.ID, name)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: user.Mention() + " has been added to the roster as " + name + "."},
		})
	case "remove":
		user := sub.Options[0].UserValue(s)
		c, cancel := storage.WithTimeout(ctx)
		defer cancel()
		_ = b.DB.DeleteMember(c, user.ID)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: user.Mention() + " has been removed from the roster."},
		})
	}
}

// /list availability|current (leader only)
func handleList(b *Bot, s *discordgo.Session, i *discordgo.InteractionCreate, ctx context.Context) {
	if !hasLeaderRole(b.Config.LeaderRoleID, i.Member) {
		ephemeralErrorRespond(s, i, "You do not have permission to use this command.")
		return
	}
	data := i.ApplicationCommandData()
	sub := data.Options[0]
	c, cancel := storage.WithTimeout(ctx)
	defer cancel()
	members, _ := b.DB.GetAllMembers(c)
	switch sub.Name {
	case "availability":
		lines := storage.FormatMembers(members, func(m storage.Member) string { return m.InGameName + " - " + m.Availability })
		embed := &discordgo.MessageEmbed{Title: "Guild Availability", Description: lines, Color: 0x00AAFF}
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}}})
	case "current":
		lines := storage.FormatMembers(members, func(m storage.Member) string {
			return m.InGameName + " - Orders: " + strconv.Itoa(m.WarOrders) + ", Lumber: " + formatNumber(m.Lumber)
		})
		embed := &discordgo.MessageEmbed{Title: "Current Guild Resources", Description: lines, Color: 0x00CC66}
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}}})
	}
}

func formatNumber(n int) string {
	in := strconv.Itoa(n)
	if len(in) <= 3 {
		return in
	}
	var b strings.Builder
	pre := len(in) % 3
	if pre == 0 {
		pre = 3
	}
	b.WriteString(in[:pre])
	for i := pre; i < len(in); i += 3 {
		b.WriteString(",")
		b.WriteString(in[i : i+3])
	}
	return b.String()
}
