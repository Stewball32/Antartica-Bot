package handlers

import (
	"context"

	"github.com/disgoorg/snowflake/v2"
)

func (h *Handler) isBotUser(_ context.Context, guildID snowflake.ID, userID snowflake.ID) bool {
	if userID == 0 {
		return false
	}

	h.botUserCacheMu.RLock()
	isBot, ok := h.botUserCache[userID]
	h.botUserCacheMu.RUnlock()
	if ok {
		return isBot
	}

	if h.client == nil {
		return false
	}

	caches := h.client.Caches()
	if member, ok := caches.Member(guildID, userID); ok {
		isBot = member.User.Bot
		h.cacheBotUser(userID, isBot)
		return isBot
	}

	member, err := h.client.Rest().GetMember(guildID, userID)
	if err != nil {
		return false
	}
	isBot = member.User.Bot
	h.cacheBotUser(userID, isBot)
	return isBot
}

func (h *Handler) cacheBotUser(userID snowflake.ID, isBot bool) {
	h.botUserCacheMu.Lock()
	h.botUserCache[userID] = isBot
	h.botUserCacheMu.Unlock()
}
