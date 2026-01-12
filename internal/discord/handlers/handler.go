package handlers

import (
	"log/slog"
	"sync"

	"antartica-bot/internal/bus"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/snowflake/v2"
)

type Handler struct {
	client bot.Client
	bus    *bus.Bus
	logger *slog.Logger

	roleToggleStore    bus.RoleToggleStore
	reactionTrackStore bus.ReactionTrackStore
	staticMessageStore bus.StaticMessageStore

	botUserCache   map[snowflake.ID]bool
	botUserCacheMu sync.RWMutex
}

func New(client bot.Client, eventBus *bus.Bus, logger *slog.Logger, roleToggleStore bus.RoleToggleStore, reactionTrackStore bus.ReactionTrackStore, staticMessageStore bus.StaticMessageStore) *Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return &Handler{
		client:             client,
		bus:                eventBus,
		logger:             logger,
		roleToggleStore:    roleToggleStore,
		reactionTrackStore: reactionTrackStore,
		staticMessageStore: staticMessageStore,
		botUserCache:       make(map[snowflake.ID]bool),
	}
}
