package handlers

import (
	"log/slog"

	"github.com/disgoorg/snowflake/v2"
)

func (h *Handler) resolveMessageAuthorID(channelID snowflake.ID, messageID snowflake.ID) snowflake.ID {
	if h == nil || h.client == nil || channelID == 0 || messageID == 0 {
		return 0
	}

	if caches := h.client.Caches(); caches != nil {
		if message, ok := caches.Message(channelID, messageID); ok {
			return message.Author.ID
		}
	}

	message, err := h.client.Rest().GetMessage(channelID, messageID)
	if err != nil || message == nil {
		if err != nil && h.logger != nil {
			h.logger.Debug(
				"message author lookup failed",
				slog.Any("err", err),
				slog.String("channel_id", channelID.String()),
				slog.String("message_id", messageID.String()),
			)
		}
		return 0
	}

	return message.Author.ID
}
