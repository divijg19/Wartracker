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

// find a reasonable role snapshot to store with the member (first matching configured role, else first role)
func firstConfiguredRole(b *Bot, guildID string, member *discordgo.Member) string {
	cfg := b.Config.GuildConfigFor(guildID)
	if cfg != nil && len(cfg.LeaderRoleIDs) > 0 {
		for _, r := range member.Roles {
			for _, id := range cfg.LeaderRoleIDs {
				if r == id {
					return r
				}
			}
		}
	}
	if b.Config.LeaderRoleID != "" {
		for _, r := range member.Roles {
			if r == b.Config.LeaderRoleID {
				return r
			}
		}
	}
	if len(member.Roles) > 0 {
		return member.Roles[0]
	}
	return ""
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
	if role := firstConfiguredRole(b, i.GuildID, i.Member); role != "" {
		_ = b.DB.UpdateMemberRole(c, i.Member.User.ID, role)
	}
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
func handleAvailability(_ *Bot, s *discordgo.Session, i *discordgo.InteractionCreate, _ context.Context) {
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
func handleHelp(_ *Bot, s *discordgo.Session, i *discordgo.InteractionCreate, _ context.Context) {
	embed := &discordgo.MessageEmbed{
		Title:       "Wartracker Bot Help",
		Description: "Slash commands overview",
		Color:       0x7289DA,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "/register in-game-name", Value: "Register or update your in-game name.", Inline: false},
			{Name: "/order amount", Value: "Set your current War Orders.", Inline: false},
			{Name: "/lumber amount", Value: "Set your current Lumber.", Inline: false},
			{Name: "/availability", Value: "Pick your 2-hour GMT window via a dropdown.", Inline: false},
			{Name: "/roster add/remove", Value: "Manage members in the roster.", Inline: false},
			{Name: "/list availability|current", Value: "Show availability or current resources.", Inline: false},
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
func handleTutorial(_ *Bot, s *discordgo.Session, i *discordgo.InteractionCreate, _ context.Context) {
	embed := &discordgo.MessageEmbed{
		Title: "Getting Started",
		Color: 0x00CC99,
		Description: strings.Join([]string{
			"1) Use /register to set your in-game name.",
			"2) Use /order and /lumber to set your current resources.",
			"3) Use /availability to choose your usual 2-hour GMT time slot.",
			"4) Use /roster to manage members and /list to share lists.",
		}, "\n"),
	}
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Flags: discordgo.MessageFlagsEphemeral},
	})
}

// /roster add|remove (leader only)
func handleRoster(b *Bot, s *discordgo.Session, i *discordgo.InteractionCreate, ctx context.Context) {
	data := i.ApplicationCommandData()
	sub := data.Options[0]
	switch sub.Name {
	case "add":
		user := sub.Options[0].UserValue(s)
		name := sub.Options[1].StringValue()
		c, cancel := storage.WithTimeout(ctx)
		defer cancel()
		_ = b.DB.UpsertMember(c, user.ID, name)
		// Try cache then API for member to snapshot a role
		var mem *discordgo.Member
		if m, err := s.State.Member(i.GuildID, user.ID); err == nil {
			mem = m
		} else if m, err := s.GuildMember(i.GuildID, user.ID); err == nil {
			mem = m
		}
		if mem != nil {
			if role := firstConfiguredRole(b, i.GuildID, mem); role != "" {
				_ = b.DB.UpdateMemberRole(c, user.ID, role)
			}
		}
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

// /syncroles [role-id]
func handleSyncRoles(b *Bot, s *discordgo.Session, i *discordgo.InteractionCreate, ctx context.Context) {
	var filterRole string
	if opts := i.ApplicationCommandData().Options; len(opts) > 0 {
		if v := opts[0].StringValue(); v != "" {
			filterRole = v
		}
	}
	// Acknowledge
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: "Syncing roles...", Flags: discordgo.MessageFlagsEphemeral},
	})

	// Sync only for the current guild to keep scope clear
	guildID := i.GuildID
	after := ""
	updated := 0
	for {
		members, err := s.GuildMembers(guildID, after, 1000)
		if err != nil || len(members) == 0 {
			break
		}
		for _, m := range members {
			if filterRole != "" {
				has := false
				for _, r := range m.Roles {
					if r == filterRole {
						has = true
						break
					}
				}
				if !has {
					continue
				}
			}
			c2, cancel := storage.WithTimeout(ctx)
			_ = b.DB.InsertMemberIfMissing(c2, m.User.ID, m.User.Username)
			if role := firstConfiguredRole(b, guildID, m); role != "" {
				_ = b.DB.UpdateMemberRole(c2, m.User.ID, role)
				updated++
			}
			cancel()
		}
		after = members[len(members)-1].User.ID
		if len(members) < 1000 {
			break
		}
	}
	// Follow-up edit
	_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Content: "Role sync complete. Updated: " + strconv.Itoa(updated)})
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
