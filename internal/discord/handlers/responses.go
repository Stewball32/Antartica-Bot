package handlers

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func respondEphemeralTone(event *events.ApplicationCommandInteractionCreate, tone EmbedTone, content string) error {
	return respondEphemeralEmbed(event, EmbedTemplate{
		Tone:        tone,
		Description: content,
	})
}

func respondEphemeralEmbed(event *events.ApplicationCommandInteractionCreate, template EmbedTemplate) error {
	embed := BuildEmbed(template)
	return event.CreateMessage(discord.MessageCreate{
		Embeds: []discord.Embed{embed},
		Flags:  discord.MessageFlagEphemeral,
	})
}
