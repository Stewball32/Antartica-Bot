package handlers

import (
	"context"

	"antartica-bot/internal/bus"

	"github.com/disgoorg/disgo/events"
)

func (h *Handler) OnGuildMessageReactionRemove(event *events.GuildMessageReactionRemove) {
	if h.bus == nil {
		return
	}
	if h.isBotUser(context.Background(), event.GuildID, event.UserID) {
		return
	}

	emojiName := ""
	if event.Emoji.Name != nil {
		emojiName = *event.Emoji.Name
	}

	h.bus.DiscordEvents <- bus.ReactionRemoved{
		GuildID:   event.GuildID,
		ChannelID: event.ChannelID,
		MessageID: event.MessageID,
		UserID:    event.UserID,
		EmojiName: emojiName,
		EmojiID:   event.Emoji.ID,
	}
}

func (h *Handler) OnGuildMessageReactionRemoveEmoji(event *events.GuildMessageReactionRemoveEmoji) {
	if h.bus == nil {
		return
	}

	emojiName := ""
	if event.Emoji.Name != nil {
		emojiName = *event.Emoji.Name
	}

	h.bus.DiscordEvents <- bus.ReactionRemovedEmoji{
		GuildID:   event.GuildID,
		ChannelID: event.ChannelID,
		MessageID: event.MessageID,
		EmojiName: emojiName,
		EmojiID:   event.Emoji.ID,
	}
}

func (h *Handler) OnGuildMessageReactionRemoveAll(event *events.GuildMessageReactionRemoveAll) {
	if h.bus == nil {
		return
	}

	h.bus.DiscordEvents <- bus.ReactionRemovedAll{
		GuildID:   event.GuildID,
		ChannelID: event.ChannelID,
		MessageID: event.MessageID,
	}
}

func (h *Handler) OnGuildMessageDelete(event *events.GuildMessageDelete) {
	if h.bus == nil {
		return
	}

	h.bus.DiscordEvents <- bus.MessageDeleted{
		GuildID:   event.GuildID,
		ChannelID: event.ChannelID,
		MessageID: event.MessageID,
	}
}
