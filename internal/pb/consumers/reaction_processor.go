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

	if event.AuthorID != 0 && event.AuthorID == event.UserID {
		return
	}

	records, err := p.findReactionRecords(ctx, event.GuildID, event.MessageID, emojiID, emojiName)
	if err != nil {
		if p.logger != nil {
			p.logger.Warn("reaction record lookup failed", slog.Any("err", err))
		}
		return
	}

	if len(records) > 0 {
		record := records[0]
		authorID := strings.TrimSpace(record.GetString("user_id"))
		if authorID == "" || authorID == event.UserID.String() {
			return
		}

		record.Set("reactions", record.GetInt("reactions")+1)
		if emojiID != "" {
			record.Set("emoji_id", emojiID)
		}
		if emojiName != "" {
			record.Set("emoji_name", emojiName)
		}

		if err := p.app.SaveWithContext(ctx, record); err != nil {
			if p.logger != nil {
				p.logger.Warn("reaction record save failed", slog.Any("err", err))
			}
		}
		return
	}

	authorID := event.AuthorID
	if authorID == 0 || authorID == event.UserID {
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
	record.Set("user_id", authorID.String())
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
	records, err := p.findReactionRecords(ctx, event.GuildID, event.MessageID, emojiID, emojiName)
	if err != nil {
		if p.logger != nil {
			p.logger.Warn("reaction record lookup failed", slog.Any("err", err))
		}
		return
	}

	if len(records) == 0 {
		return
	}

	record := records[0]
	authorID := strings.TrimSpace(record.GetString("user_id"))
	if authorID == "" || authorID == event.UserID.String() {
		return
	}

	next := record.GetInt("reactions") - 1
	if next <= 0 {
		if err := p.app.DeleteWithContext(ctx, record); err != nil && p.logger != nil {
			p.logger.Warn("reaction record delete failed", slog.Any("err", err))
		}
		return
	}

	record.Set("reactions", next)
	if err := p.app.SaveWithContext(ctx, record); err != nil && p.logger != nil {
		p.logger.Warn("reaction record save failed", slog.Any("err", err))
	}
}

func (p *ReactionProcessor) HandleReactionRemoveEmoji(ctx context.Context, event bus.ReactionRemovedEmoji) {
	if p == nil || p.app == nil {
		return
	}

	emojiID, emojiName := normalizeEmoji(event.EmojiID, event.EmojiName)
	records, err := p.findReactionRecords(ctx, event.GuildID, event.MessageID, emojiID, emojiName)
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

	records, err := p.findReactionRecords(ctx, event.GuildID, event.MessageID, "", "")
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

	records, err := p.findReactionRecords(ctx, event.GuildID, event.MessageID, "", "")
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

func (p *ReactionProcessor) findReactionRecords(_ context.Context, guildID snowflake.ID, messageID snowflake.ID, emojiID string, emojiName string) ([]*core.Record, error) {
	if p == nil || p.app == nil {
		return nil, nil
	}

	filter := dbx.HashExp{
		"guild_id":   guildID.String(),
		"message_id": messageID.String(),
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
