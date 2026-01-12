package commands

import "github.com/disgoorg/disgo/discord"

// Builder returns a command definition to register with Discord.
type Builder func() discord.ApplicationCommandCreate

var builders []Builder

// Register adds a command builder. Use one file per command and register in init.
func Register(builder Builder) {
	if builder == nil {
		return
	}
	builders = append(builders, builder)
}

// All returns all registered commands.
func All() []discord.ApplicationCommandCreate {
	commands := make([]discord.ApplicationCommandCreate, 0, len(builders))
	for _, builder := range builders {
		if builder == nil {
			continue
		}
		commands = append(commands, builder())
	}
	return commands
}
