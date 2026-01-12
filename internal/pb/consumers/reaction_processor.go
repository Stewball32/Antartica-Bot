package consumers

import (
	"context"
	"log/slog"
	"strings"

	"antartica-bot/internal/bus"

	"github.com/disgoorg/snowflake/v2"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type ReactionProcessor struct {
	app    core.App
	logger *slog.Logger
}

func NewReactionProcessor(app core.App, logger *slog.Logger) *ReactionProcessor {
	if logger == nil && app != nil {
		logger = app.Logger()
	}
	return &ReactionProcessor{
		app:    app,
		logger: logger,
	}
}

func (p *ReactionProcessor) HandleReactionAdd(ctx context.Context, event bus.ReactionAdded) {
	if p == nil || p.app == nil {
		return
	}

	emojiID, emojiName := normalizeEmoji(event.EmojiID, event.EmojiName)
	tracked, err := p.isTrackedReaction(ctx, event.GuildID, emojiID, emojiName)
	if err != nil {
		if p.logger != nil {
			p.logger.Warn("reaction track lookup failed", slog.Any("err", err))
		}
		return
	}
	if !tracked {
		return
	}

	if exists, err := p.reactionRecordExists(ctx, event.GuildID, event.MessageID, event.UserID, emojiID, emojiName); err != nil {
		if p.logger != nil {
			p.logger.Warn("reaction record lookup failed", slog.Any("err", err))
		}
		return
	} else if exists {
		return
	}

	collection, err := p.app.FindCollectionByNameOrId("reaction_records")
	if err != nil {
		if p.logger != nil {
			p.logger.Warn("reaction records collection missing", slog.Any("err", err))
		}
		return
	}

	record := core.NewRecord(collection)
	record.Set("guild_id", event.GuildID.String())
	record.Set("message_id", event.MessageID.String())
	record.Set("user_id", event.UserID.String())
	record.Set("emoji_id", emojiID)
	record.Set("emoji_name", emojiName)
	record.Set("reactions", 1)

	if err := p.app.SaveWithContext(ctx, record); err != nil {
		if p.logger != nil {
			p.logger.Warn("reaction record save failed", slog.Any("err", err))
		}
	}
}

func (p *ReactionProcessor) HandleReactionRemove(ctx context.Context, event bus.ReactionRemoved) {
	if p == nil || p.app == nil {
		return
	}

	emojiID, emojiName := normalizeEmoji(event.EmojiID, event.EmojiName)
	records, err := p.findReactionRecords(ctx, event.GuildID, event.MessageID, event.UserID, emojiID, emojiName)
	if err != nil {
		if p.logger != nil {
			p.logger.Warn("reaction record lookup failed", slog.Any("err", err))
		}
		return
	}

	for _, record := range records {
		if err := p.app.DeleteWithContext(ctx, record); err != nil && p.logger != nil {
			p.logger.Warn("reaction record delete failed", slog.Any("err", err))
		}
	}
}

func (p *ReactionProcessor) HandleReactionRemoveEmoji(ctx context.Context, event bus.ReactionRemovedEmoji) {
	if p == nil || p.app == nil {
		return
	}

	emojiID, emojiName := normalizeEmoji(event.EmojiID, event.EmojiName)
	records, err := p.findReactionRecords(ctx, event.GuildID, event.MessageID, 0, emojiID, emojiName)
	if err != nil {
		if p.logger != nil {
			p.logger.Warn("reaction record lookup failed", slog.Any("err", err))
		}
		return
	}

	for _, record := range records {
		if err := p.app.DeleteWithContext(ctx, record); err != nil && p.logger != nil {
			p.logger.Warn("reaction record delete failed", slog.Any("err", err))
		}
	}
}

func (p *ReactionProcessor) HandleReactionRemoveAll(ctx context.Context, event bus.ReactionRemovedAll) {
	if p == nil || p.app == nil {
		return
	}

	records, err := p.findReactionRecords(ctx, event.GuildID, event.MessageID, 0, "", "")
	if err != nil {
		if p.logger != nil {
			p.logger.Warn("reaction record lookup failed", slog.Any("err", err))
		}
		return
	}

	for _, record := range records {
		if err := p.app.DeleteWithContext(ctx, record); err != nil && p.logger != nil {
			p.logger.Warn("reaction record delete failed", slog.Any("err", err))
		}
	}
}

func (p *ReactionProcessor) HandleMessageDeleted(ctx context.Context, event bus.MessageDeleted) {
	if p == nil || p.app == nil {
		return
	}

	records, err := p.findReactionRecords(ctx, event.GuildID, event.MessageID, 0, "", "")
	if err != nil {
		if p.logger != nil {
			p.logger.Warn("reaction record lookup failed", slog.Any("err", err))
		}
		return
	}

	for _, record := range records {
		if err := p.app.DeleteWithContext(ctx, record); err != nil && p.logger != nil {
			p.logger.Warn("reaction record delete failed", slog.Any("err", err))
		}
	}
}

func (p *ReactionProcessor) isTrackedReaction(_ context.Context, guildID snowflake.ID, emojiID string, emojiName string) (bool, error) {
	if emojiID == "" && emojiName == "" {
		return false, nil
	}

	records, err := p.app.FindAllRecords("reaction_tracks", reactionTrackFilter(guildID, emojiID, emojiName))
	if err != nil {
		return false, err
	}

	return len(records) > 0, nil
}

func (p *ReactionProcessor) reactionRecordExists(ctx context.Context, guildID snowflake.ID, messageID snowflake.ID, userID snowflake.ID, emojiID string, emojiName string) (bool, error) {
	records, err := p.findReactionRecords(ctx, guildID, messageID, userID, emojiID, emojiName)
	if err != nil {
		return false, err
	}
	return len(records) > 0, nil
}

func (p *ReactionProcessor) findReactionRecords(_ context.Context, guildID snowflake.ID, messageID snowflake.ID, userID snowflake.ID, emojiID string, emojiName string) ([]*core.Record, error) {
	if p == nil || p.app == nil {
		return nil, nil
	}

	filter := dbx.HashExp{
		"guild_id":   guildID.String(),
		"message_id": messageID.String(),
	}
	if userID != 0 {
		filter["user_id"] = userID.String()
	}
	if emojiID != "" {
		filter["emoji_id"] = emojiID
	} else if emojiName != "" {
		filter["emoji_name"] = emojiName
	}

	return p.app.FindAllRecords("reaction_records", filter)
}

func normalizeEmoji(emojiID *snowflake.ID, emojiName string) (string, string) {
	name := strings.TrimSpace(emojiName)
	if emojiID != nil {
		return emojiID.String(), name
	}
	return "", name
}

func reactionTrackFilter(guildID snowflake.ID, emojiID string, emojiName string) dbx.HashExp {
	filter := dbx.HashExp{
		"guild_id": guildID.String(),
	}
	emojiID = strings.TrimSpace(emojiID)
	if emojiID != "" {
		filter["emoji_id"] = emojiID
		return filter
	}
	filter["emoji_name"] = strings.TrimSpace(emojiName)
	return filter
}
