package storage

// Member maps to the members table.
type Member struct {
	DiscordID    string
	InGameName   string
	WarOrders    int
	Lumber       int
	Availability string
	GuildRoleID  string
}
