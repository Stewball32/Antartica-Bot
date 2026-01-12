package actions

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"antartica-bot/internal/bus"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
)

func StartActionWorker(ctx context.Context, client bot.Client, eventBus *bus.Bus, logger *slog.Logger) {
	if eventBus == nil {
		return
	}
	if logger == nil {
		logger = slog.Default()
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case action, ok := <-eventBus.DiscordActions:
				if !ok {
					return
				}
				handleAction(ctx, client, logger, action)
			}
		}
	}()
}

func handleAction(ctx context.Context, client bot.Client, logger *slog.Logger, action bus.DiscordAction) {
	switch payload := action.(type) {
	case bus.SendMessage:
		if payload.Content == "" {
			return
		}
		_, err := client.Rest().CreateMessage(
			payload.ChannelID,
			discord.NewMessageCreateBuilder().SetContent(payload.Content).Build(),
		)
		if err != nil {
			logger.Error(
				"discord send message failed",
				slog.Any("err", err),
				slog.String("channel_id", payload.ChannelID.String()),
			)
		}
	case bus.EditMessage:
		if payload.ChannelID == 0 || payload.MessageID == 0 {
			return
		}
		update := discord.MessageUpdate{}
		if payload.Content != "" {
			update.Content = &payload.Content
		}
		if len(payload.Embeds) > 0 {
			update.Embeds = &payload.Embeds
		}
		_, err := client.Rest().UpdateMessage(payload.ChannelID, payload.MessageID, update)
		if err != nil {
			logger.Error(
				"discord update message failed",
				slog.Any("err", err),
				slog.String("channel_id", payload.ChannelID.String()),
				slog.String("message_id", payload.MessageID.String()),
			)
		}
	case bus.LogEvent:
		logEventToConsole(ctx, logger, payload)
	default:
		logger.Warn("unknown discord action", slog.String("type", fmt.Sprintf("%T", action)))
	}
}

func logEventToConsole(ctx context.Context, logger *slog.Logger, event bus.LogEvent) {
	if logger == nil {
		return
	}

	level := slog.LevelInfo
	switch strings.ToLower(string(event.Level)) {
	case string(bus.LogDebug):
		level = slog.LevelDebug
	case string(bus.LogInfo):
		level = slog.LevelInfo
	case string(bus.LogWarn):
		level = slog.LevelWarn
	case string(bus.LogError):
		level = slog.LevelError
	}

	message := strings.TrimSpace(event.Title)
	if message == "" {
		message = defaultLogMessage(event)
	}

	args := []any{
		"guild_id", event.GuildID.String(),
		"category", strings.TrimSpace(event.Category),
		"level", string(event.Level),
	}
	if desc := strings.TrimSpace(event.Description); desc != "" {
		args = append(args, "description", desc)
	}
	if !event.Timestamp.IsZero() {
		args = append(args, "timestamp", event.Timestamp)
	}
	for _, field := range event.Fields {
		name := strings.TrimSpace(field.Name)
		value := strings.TrimSpace(field.Value)
		if name == "" || value == "" {
			continue
		}
		args = append(args, name, value)
	}

	logger.Log(ctx, level, message, args...)
}

func defaultLogMessage(event bus.LogEvent) string {
	level := strings.ToUpper(strings.TrimSpace(string(event.Level)))
	category := strings.TrimSpace(event.Category)
	if level == "" && category == "" {
		return "Log Event"
	}
	if level == "" {
		return category
	}
	if category == "" {
		return level
	}
	return fmt.Sprintf("%s | %s", level, category)
}
