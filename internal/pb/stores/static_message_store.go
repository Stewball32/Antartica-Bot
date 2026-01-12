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

type StaticMessageStore struct {
	app    core.App
	logger *slog.Logger
}

func NewStaticMessageStore(app core.App, logger *slog.Logger) *StaticMessageStore {
	if logger == nil && app != nil {
		logger = app.Logger()
	}
	return &StaticMessageStore{
		app:    app,
		logger: logger,
	}
}

func (s *StaticMessageStore) CreateStaticMessage(ctx context.Context, guildID snowflake.ID, channelID snowflake.ID, messageID snowflake.ID, messageType string, config string, update string) error {
	if s == nil || s.app == nil {
		return errors.New("static message store is not configured")
	}

	collection, err := s.app.FindCollectionByNameOrId("static_messages")
	if err != nil {
		return err
	}

	record := core.NewRecord(collection)
	record.Set("guild_id", guildID.String())
	record.Set("channel_id", channelID.String())
	record.Set("message_id", messageID.String())
	record.Set("type", strings.TrimSpace(messageType))
	record.Set("config", strings.TrimSpace(config))
	record.Set("update", strings.TrimSpace(update))

	if err := s.app.SaveWithContext(ctx, record); err != nil {
		return err
	}

	if s.logger != nil {
		s.logger.Info(
			"static message added",
			slog.String("guild_id", guildID.String()),
			slog.String("message_id", messageID.String()),
			slog.String("type", messageType),
			slog.String("update", update),
		)
	}

	return nil
}

func (s *StaticMessageStore) RemoveStaticMessage(ctx context.Context, guildID snowflake.ID, messageID snowflake.ID) (int, error) {
	if s == nil || s.app == nil {
		return 0, errors.New("static message store is not configured")
	}

	records, err := s.app.FindAllRecords("static_messages", dbx.HashExp{
		"guild_id":   guildID.String(),
		"message_id": messageID.String(),
	})
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
			"static message removed",
			slog.String("guild_id", guildID.String()),
			slog.String("message_id", messageID.String()),
			slog.Int("count", deleted),
		)
	}

	return deleted, nil
}

func (s *StaticMessageStore) ListStaticMessages(ctx context.Context, guildID snowflake.ID) ([]bus.StaticMessage, error) {
	if s == nil || s.app == nil {
		return nil, errors.New("static message store is not configured")
	}

	records, err := s.app.FindAllRecords("static_messages", dbx.HashExp{
		"guild_id": guildID.String(),
	})
	if err != nil {
		return nil, err
	}

	messages := make([]bus.StaticMessage, 0, len(records))
	for _, record := range records {
		channelID, err := snowflake.Parse(record.GetString("channel_id"))
		if err != nil {
			continue
		}
		messageID, err := snowflake.Parse(record.GetString("message_id"))
		if err != nil {
			continue
		}
		messages = append(messages, bus.StaticMessage{
			ChannelID: channelID,
			MessageID: messageID,
			Type:      strings.TrimSpace(record.GetString("type")),
			Config:    strings.TrimSpace(record.GetString("config")),
			Update:    strings.TrimSpace(record.GetString("update")),
		})
	}

	return messages, nil
}

var _ bus.StaticMessageStore = (*StaticMessageStore)(nil)
