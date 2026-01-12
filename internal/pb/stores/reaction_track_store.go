package stores

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"antartica-bot/internal/bus"

	"github.com/disgoorg/snowflake/v2"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type ReactionTrackStore struct {
	app    core.App
	logger *slog.Logger
}

func NewReactionTrackStore(app core.App, logger *slog.Logger) *ReactionTrackStore {
	if logger == nil && app != nil {
		logger = app.Logger()
	}
	return &ReactionTrackStore{
		app:    app,
		logger: logger,
	}
}

func (s *ReactionTrackStore) UpsertReactionTrack(ctx context.Context, guildID snowflake.ID, emojiID string, emojiName string, title string, description string) (bool, error) {
	if s == nil || s.app == nil {
		return false, errors.New("reaction track store is not configured")
	}

	emojiID = strings.TrimSpace(emojiID)
	emojiName = strings.TrimSpace(emojiName)
	if emojiID == "" && emojiName == "" {
		return false, errors.New("emoji id or name is required")
	}

	records, err := s.app.FindAllRecords("reaction_tracks", reactionTrackFilter(guildID, emojiID, emojiName))
	if err != nil {
		return false, err
	}

	if len(records) > 0 {
		record := records[0]
		record.Set("title", title)
		record.Set("description", description)
		if emojiID != "" {
			record.Set("emoji_id", emojiID)
		}
		if emojiName != "" {
			record.Set("emoji_name", emojiName)
		}
		if err := s.app.SaveWithContext(ctx, record); err != nil {
			return false, err
		}
		return false, nil
	}

	collection, err := s.app.FindCollectionByNameOrId("reaction_tracks")
	if err != nil {
		return false, err
	}

	record := core.NewRecord(collection)
	record.Set("guild_id", guildID.String())
	record.Set("emoji_id", emojiID)
	record.Set("emoji_name", emojiName)
	record.Set("title", title)
	record.Set("description", description)

	if err := s.app.SaveWithContext(ctx, record); err != nil {
		return false, err
	}

	if s.logger != nil {
		s.logger.Info(
			"reaction track added",
			slog.String("guild_id", guildID.String()),
			slog.String("emoji_id", emojiID),
			slog.String("emoji_name", emojiName),
		)
	}

	return true, nil
}

func (s *ReactionTrackStore) RemoveReactionTrack(ctx context.Context, guildID snowflake.ID, emojiID string, emojiName string) (int, error) {
	if s == nil || s.app == nil {
		return 0, errors.New("reaction track store is not configured")
	}

	records, err := s.app.FindAllRecords("reaction_tracks", reactionTrackFilter(guildID, emojiID, emojiName))
	if err != nil {
		return 0, err
	}
	if len(records) == 0 {
		return 0, nil
	}

	deleted := 0
	for _, record := range records {
		if err := s.app.DeleteWithContext(ctx, record); err != nil {
			return deleted, err
		}
		deleted++
	}

	if s.logger != nil {
		s.logger.Info(
			"reaction track removed",
			slog.String("guild_id", guildID.String()),
			slog.String("emoji_id", strings.TrimSpace(emojiID)),
			slog.String("emoji_name", strings.TrimSpace(emojiName)),
			slog.Int("count", deleted),
		)
	}

	return deleted, nil
}

func (s *ReactionTrackStore) ListReactionTracks(ctx context.Context, guildID snowflake.ID) ([]bus.ReactionTrack, error) {
	if s == nil || s.app == nil {
		return nil, errors.New("reaction track store is not configured")
	}

	records, err := s.app.FindAllRecords("reaction_tracks", dbx.HashExp{
		"guild_id": guildID.String(),
	})
	if err != nil {
		return nil, err
	}

	tracks := make([]bus.ReactionTrack, 0, len(records))
	for _, record := range records {
		tracks = append(tracks, bus.ReactionTrack{
			EmojiID:     strings.TrimSpace(record.GetString("emoji_id")),
			EmojiName:   strings.TrimSpace(record.GetString("emoji_name")),
			Title:       strings.TrimSpace(record.GetString("title")),
			Description: strings.TrimSpace(record.GetString("description")),
		})
	}

	return tracks, nil
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

var _ bus.ReactionTrackStore = (*ReactionTrackStore)(nil)
