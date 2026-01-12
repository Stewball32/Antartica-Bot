package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/json"
)

const BotCommandName = "bot"

func init() {
	Register(BotCommand)
}

func BotCommand() discord.ApplicationCommandCreate {
	admin := json.NewNullable(discord.PermissionAdministrator)
	return discord.SlashCommandCreate{
		Name:                     BotCommandName,
		Description:              "Manage the bot profile and presence",
		Contexts:                 []discord.InteractionContextType{discord.InteractionContextTypeGuild},
		DefaultMemberPermissions: &admin,
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionSubCommand{
				Name:        "name",
				Description: "Update the bot username",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{
						Name:        "name",
						Description: "New bot username",
						Required:    true,
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:        "avatar",
				Description: "Update the bot avatar",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionAttachment{
						Name:        "attachment",
						Description: "Image attachment for the avatar",
					},
					discord.ApplicationCommandOptionString{
						Name:        "url",
						Description: "Image URL for the avatar",
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:        "banner",
				Description: "Update the bot banner",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionAttachment{
						Name:        "attachment",
						Description: "Image attachment for the banner",
					},
					discord.ApplicationCommandOptionString{
						Name:        "url",
						Description: "Image URL for the banner",
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:        "about",
				Description: "Update the bot profile description",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{
						Name:        "text",
						Description: "New description text",
						Required:    true,
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:        "status",
				Description: "Update the bot status (blank clears to online)",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{
						Name:        "status",
						Description: "New status",
						Choices: []discord.ApplicationCommandOptionChoiceString{
							{Name: "online", Value: "online"},
							{Name: "idle", Value: "idle"},
							{Name: "dnd", Value: "dnd"},
							{Name: "invisible", Value: "invisible"},
							{Name: "offline", Value: "offline"},
						},
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:        "activity",
				Description: "Update the bot activity (blank clears)",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{
						Name:        "type",
						Description: "Activity type",
						Choices: []discord.ApplicationCommandOptionChoiceString{
							{Name: "playing", Value: "playing"},
							{Name: "listening", Value: "listening"},
							{Name: "watching", Value: "watching"},
							{Name: "competing", Value: "competing"},
							{Name: "custom", Value: "custom"},
						},
					},
					discord.ApplicationCommandOptionString{
						Name:        "text",
						Description: "Activity text",
					},
					discord.ApplicationCommandOptionString{
						Name:        "emoji",
						Description: "Emoji for custom activity (unicode or <:name:id>)",
					},
				},
			},
		},
	}
}
