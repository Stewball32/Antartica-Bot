package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/json"
)

const RoleToggleCommandName = "role"

func init() {
	Register(RoleToggleCommand)
}

func RoleToggleCommand() discord.ApplicationCommandCreate {
	manageRoles := json.NewNullable(discord.PermissionManageRoles)
	return discord.SlashCommandCreate{
		Name:                     RoleToggleCommandName,
		Description:              "Configure self-assignable roles",
		Contexts:                 []discord.InteractionContextType{discord.InteractionContextTypeGuild},
		DefaultMemberPermissions: &manageRoles,
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionSubCommand{
				Name:        "add",
				Description: "Add a role to the self-toggle list",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionRole{
						Name:        "role",
						Description: "Role to add",
						Required:    true,
					},
					discord.ApplicationCommandOptionString{
						Name:        "description",
						Description: "Optional description for the role",
					},
					discord.ApplicationCommandOptionString{
						Name:        "permissions",
						Description: "Permission integer required to toggle (default: 0)",
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:        "remove",
				Description: "Remove a role from the self-toggle list",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionRole{
						Name:        "role",
						Description: "Role to remove",
						Required:    true,
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:        "message-create",
				Description: "Create a static role list message",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionChannel{
						Name:         "channel",
						Description:  "Channel to post the message in",
						Required:     true,
						ChannelTypes: []discord.ChannelType{discord.ChannelTypeGuildText, discord.ChannelTypeGuildNews},
					},
					discord.ApplicationCommandOptionString{
						Name:        "title",
						Description: "Optional title override",
					},
					discord.ApplicationCommandOptionString{
						Name:        "description",
						Description: "Optional description override",
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:        "message-remove",
				Description: "Stop updating a static role list message",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{
						Name:        "message_id",
						Description: "Message ID to stop updating",
						Required:    true,
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:        "message-list",
				Description: "List static role list messages",
			},
		},
	}
}
