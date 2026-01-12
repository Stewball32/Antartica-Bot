package handlers

import (
	"context"

	"antartica-bot/internal/bus"

	"github.com/disgoorg/disgo/events"
)

func (h *Handler) OnGuildMessageReactionAdd(event *events.GuildMessageReactionAdd) {
	if h.bus == nil {
		return
	}

	if event.Member.User.ID != 0 && event.Member.User.ID == event.UserID {
		if event.Member.User.Bot {
			h.cacheBotUser(event.UserID, true)
			return
		}
		h.cacheBotUser(event.UserID, false)
	} else if h.isBotUser(context.Background(), event.GuildID, event.UserID) {
		return
	}

	emojiName := ""
	if event.Emoji.Name != nil {
		emojiName = *event.Emoji.Name
	}

	authorID := h.resolveMessageAuthorID(event.ChannelID, event.MessageID)

	h.bus.DiscordEvents <- bus.ReactionAdded{
		GuildID:   event.GuildID,
		ChannelID: event.ChannelID,
		MessageID: event.MessageID,
		UserID:    event.UserID,
		AuthorID:  authorID,
		EmojiName: emojiName,
		EmojiID:   event.Emoji.ID,
	}
}

func (h *Handler) OnInteractionCreate(event *events.InteractionCreate) {
	if h.bus == nil {
		return
	}

	interaction := event.Interaction
	h.bus.DiscordEvents <- bus.InteractionReceived{
		InteractionID:   interaction.ID(),
		InteractionType: interaction.Type(),
		GuildID:         interaction.GuildID(),
		ChannelID:       interaction.ChannelID(),
		UserID:          interaction.User().ID,
	}
}
