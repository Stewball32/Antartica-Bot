package hooks

import (
	"context"
	"log/slog"
	"strings"

	"antartica-bot/internal/bus"
	"antartica-bot/internal/pb/messages"
	"antartica-bot/internal/pb/schema"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterHooks(app *pocketbase.PocketBase, eventBus *bus.Bus, logger *slog.Logger) {
	if logger == nil {
		logger = app.Logger()
	}

	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		return schema.Ensure(e.App, logger)
	})

	app.OnRecordAfterCreateSuccess().BindFunc(func(e *core.RecordEvent) error {
		logger.Debug(
			"pocketbase record created",
			slog.String("collection", e.Record.Collection().Name),
			slog.String("record_id", e.Record.Id),
		)

		if e.Record.Collection().Name == "reaction_records" {
			if err := messages.UpdateReactionLeaderboard(context.Background(), e.App, eventBus, logger, e.Record, 0); err != nil {
				logger.Warn("reaction leaderboard update failed", slog.Any("err", err))
			}
		}

		if e.Record.Collection().Name == "role_toggles" && eventBus != nil {
			if err := messages.EnqueueRoleToggleUpdates(context.Background(), e.App, eventBus, logger, e.Record.GetString("guild_id")); err != nil {
				logger.Warn("role toggle message update failed", slog.Any("err", err))
			}
		}

		if e.Record.Collection().Name == "static_messages" && eventBus != nil {
			if strings.TrimSpace(e.Record.GetString("update")) != messages.StaticMessageUpdateInstant {
				return e.Next()
			}

			messageType := strings.TrimSpace(e.Record.GetString("type"))
			switch messageType {
			case messages.StaticMessageTypeLeaderboard:
				config, err := messages.ParseLeaderboardConfig(e.Record.GetString("config"))
				if err != nil {
					logger.Warn("static message config invalid", slog.Any("err", err))
				} else if err := messages.EnqueueLeaderboardUpdates(context.Background(), e.App, eventBus, logger, e.Record.GetString("guild_id"), config.EmojiID, config.EmojiName); err != nil {
					logger.Warn("static message update failed", slog.Any("err", err))
				}
			case messages.StaticMessageTypeRoleToggles:
				if err := messages.EnqueueRoleToggleUpdates(context.Background(), e.App, eventBus, logger, e.Record.GetString("guild_id")); err != nil {
					logger.Warn("static message update failed", slog.Any("err", err))
				}
			}
		}

		return e.Next()
	})

	app.OnRecordAfterUpdateSuccess("role_toggles").BindFunc(func(e *core.RecordEvent) error {
		if eventBus != nil {
			if err := messages.EnqueueRoleToggleUpdates(context.Background(), e.App, eventBus, logger, e.Record.GetString("guild_id")); err != nil {
				logger.Warn("role toggle message update failed", slog.Any("err", err))
			}
		}
		return e.Next()
	})

	app.OnRecordAfterDeleteSuccess("reaction_records").BindFunc(func(e *core.RecordEvent) error {
		if err := messages.UpdateReactionLeaderboard(context.Background(), e.App, eventBus, logger, e.Record, -1); err != nil {
			logger.Warn("reaction leaderboard update failed", slog.Any("err", err))
		}
		return e.Next()
	})

	app.OnRecordAfterDeleteSuccess("role_toggles").BindFunc(func(e *core.RecordEvent) error {
		if eventBus != nil {
			if err := messages.EnqueueRoleToggleUpdates(context.Background(), e.App, eventBus, logger, e.Record.GetString("guild_id")); err != nil {
				logger.Warn("role toggle message update failed", slog.Any("err", err))
			}
		}
		return e.Next()
	})
}
