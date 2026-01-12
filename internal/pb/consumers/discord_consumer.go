package consumers

import (
	"context"
	"fmt"
	"log/slog"

	"antartica-bot/internal/bus"

	"github.com/pocketbase/pocketbase"
)

type DiscordConsumer struct {
	app    *pocketbase.PocketBase
	bus    *bus.Bus
	logger *slog.Logger

	reactions *ReactionProcessor
}

func StartDiscordConsumer(ctx context.Context, app *pocketbase.PocketBase, eventBus *bus.Bus, logger *slog.Logger) {
	if eventBus == nil {
		return
	}
	if logger == nil {
		logger = app.Logger()
	}

	consumer := &DiscordConsumer{
		app:       app,
		bus:       eventBus,
		logger:    logger,
		reactions: NewReactionProcessor(app, logger),
	}

	go consumer.run(ctx)
}

func (c *DiscordConsumer) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-c.bus.DiscordEvents:
			if !ok {
				return
			}
			c.handle(event)
		}
	}
}

func (c *DiscordConsumer) handle(event bus.DiscordEvent) {
	switch payload := event.(type) {
	case bus.ReactionAdded:
		if c.reactions != nil {
			c.reactions.HandleReactionAdd(context.Background(), payload)
		}
	case bus.ReactionRemoved:
		if c.reactions != nil {
			c.reactions.HandleReactionRemove(context.Background(), payload)
		}
	case bus.ReactionRemovedEmoji:
		if c.reactions != nil {
			c.reactions.HandleReactionRemoveEmoji(context.Background(), payload)
		}
	case bus.ReactionRemovedAll:
		if c.reactions != nil {
			c.reactions.HandleReactionRemoveAll(context.Background(), payload)
		}
	case bus.MessageDeleted:
		if c.reactions != nil {
			c.reactions.HandleMessageDeleted(context.Background(), payload)
		}
	case bus.InteractionReceived:
		c.logger.Debug(
			"discord interaction received",
			slog.String("interaction_id", payload.InteractionID.String()),
			slog.Int("type", int(payload.InteractionType)),
		)
	default:
		c.logger.Warn("unknown discord event", slog.String("type", fmt.Sprintf("%T", event)))
	}

	// TODO: use c.app.Dao() to persist events to PocketBase collections or invoke domain services.
}
