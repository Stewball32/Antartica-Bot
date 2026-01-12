package commands

import "github.com/disgoorg/disgo/discord"

const RoleSelfCommandName = "toggle-role"

func init() {
	Register(RoleSelfCommand)
}

func RoleSelfCommand() discord.ApplicationCommandCreate {
	return discord.SlashCommandCreate{
		Name:        RoleSelfCommandName,
		Description: "Toggle self-assignable roles",
		Contexts:    []discord.InteractionContextType{discord.InteractionContextTypeGuild},
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionRole{
				Name:        "role",
				Description: "Role to toggle (leave empty to list)",
			},
		},
	}
}
