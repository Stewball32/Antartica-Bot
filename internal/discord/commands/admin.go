package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/json"
)

const ReactionCommandName = "reaction"

func init() {
	Register(ReactionCommand)
}

func ReactionCommand() discord.ApplicationCommandCreate {
	manageGuild := json.NewNullable(discord.PermissionManageGuild)
	return discord.SlashCommandCreate{
		Name:                     ReactionCommandName,
		Description:              "Reaction tracking tools",
		Contexts:                 []discord.InteractionContextType{discord.InteractionContextTypeGuild},
		DefaultMemberPermissions: &manageGuild,
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionSubCommand{
				Name:        "add",
				Description: "Track an emoji for leaderboard stats",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{
						Name:        "emoji",
						Description: "Emoji to track (unicode or custom like <:name:id>)",
						Required:    true,
					},
					discord.ApplicationCommandOptionString{
						Name:        "title",
						Description: "Display title for this emoji",
						Required:    true,
					},
					discord.ApplicationCommandOptionString{
						Name:        "description",
						Description: "Optional description",
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:        "remove",
				Description: "Stop tracking an emoji",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{
						Name:        "emoji",
						Description: "Emoji to remove",
						Required:    true,
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:        "list",
				Description: "List tracked emojis",
			},
			discord.ApplicationCommandOptionSubCommandGroup{
				Name:        "leaderboard",
				Description: "Manage leaderboard messages",
				Options: []discord.ApplicationCommandOptionSubCommand{
					{
						Name:        "create",
						Description: "Create a leaderboard message",
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionChannel{
								Name:         "channel",
								Description:  "Channel to post the message in",
								Required:     true,
								ChannelTypes: []discord.ChannelType{discord.ChannelTypeGuildText, discord.ChannelTypeGuildNews},
							},
							discord.ApplicationCommandOptionString{
								Name:        "emoji",
								Description: "Emoji to show leaderboard for",
								Required:    true,
							},
							discord.ApplicationCommandOptionString{
								Name:        "update",
								Description: "How often to update the message",
								Required:    true,
								Choices: []discord.ApplicationCommandOptionChoiceString{
									{Name: "instant", Value: "instant"},
									{Name: "hourly", Value: "hourly"},
									{Name: "daily", Value: "daily"},
								},
							},
							discord.ApplicationCommandOptionInt{
								Name:        "top",
								Description: "How many users to show",
							},
							discord.ApplicationCommandOptionString{
								Name:        "title",
								Description: "Optional title override",
							},
						},
					},
					{
						Name:        "remove",
						Description: "Remove a leaderboard message",
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionString{
								Name:        "message_id",
								Description: "Message ID to stop updating",
								Required:    true,
							},
						},
					},
					{
						Name:        "list",
						Description: "List leaderboard messages",
					},
				},
			},
		},
	}
}
