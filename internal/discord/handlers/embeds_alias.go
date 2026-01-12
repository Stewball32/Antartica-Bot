package handlers

import (
	"antartica-bot/internal/discord/embeds"

	"github.com/disgoorg/disgo/discord"
)

type EmbedTone = embeds.EmbedTone
type EmbedTemplate = embeds.EmbedTemplate

const (
	EmbedInfo     = embeds.EmbedInfo
	EmbedSuccess  = embeds.EmbedSuccess
	EmbedDecline  = embeds.EmbedDecline
	EmbedQuestion = embeds.EmbedQuestion
	EmbedError    = embeds.EmbedError
	EmbedWarn     = embeds.EmbedWarn
	EmbedDebug    = embeds.EmbedDebug
	EmbedNeutral  = embeds.EmbedNeutral
)

func BuildEmbed(template EmbedTemplate) discord.Embed {
	return embeds.BuildEmbed(template)
}
