package messages

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"antartica-bot/internal/bus"
	discordembed "antartica-bot/internal/discord/embeds"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	StaticMessageTypeLeaderboard = "reaction_leaderboard"
	StaticMessageUpdateInstant   = "instant"
)

type leaderboardConfig struct {
	EmojiID   string `json:"emoji_id,omitempty"`
	EmojiName string `json:"emoji_name,omitempty"`
	Top       int    `json:"top,omitempty"`
	Title     string `json:"title,omitempty"`
}

type leaderboardEntry struct {
	UserID    string
	Reactions int
}

func UpdateReactionLeaderboard(ctx context.Context, app core.App, eventBus *bus.Bus, logger *slog.Logger, record *core.Record, delta int) error {
	if record == nil || app == nil {
		return nil
	}

	guildID := strings.TrimSpace(record.GetString("guild_id"))
	userID := strings.TrimSpace(record.GetString("user_id"))
	emojiID := strings.TrimSpace(record.GetString("emoji_id"))
	emojiName := strings.TrimSpace(record.GetString("emoji_name"))
	if guildID == "" || userID == "" {
		return nil
	}

	if delta == 0 {
		delta = record.GetInt("reactions")
	}
	if delta == 0 {
		delta = 1
	}

	filter := dbx.HashExp{
		"guild_id": guildID,
		"user_id":  userID,
	}
	if emojiID != "" {
		filter["emoji_id"] = emojiID
	} else if emojiName != "" {
		filter["emoji_name"] = emojiName
	} else {
		return nil
	}

	records, err := app.FindAllRecords("reaction_leaderboard", filter)
	if err != nil {
		return err
	}

	if len(records) > 0 {
		entry := records[0]
		current := entry.GetInt("reactions")
		next := current + delta
		if next <= 0 {
			if err := app.DeleteWithContext(ctx, entry); err != nil {
				return err
			}
		} else {
			entry.Set("reactions", next)
			if emojiName != "" {
				entry.Set("emoji_name", emojiName)
			}
			if emojiID != "" {
				entry.Set("emoji_id", emojiID)
			}
			if err := app.SaveWithContext(ctx, entry); err != nil {
				return err
			}
		}
	} else if delta > 0 {
		collection, err := app.FindCollectionByNameOrId("reaction_leaderboard")
		if err != nil {
			return err
		}
		entry := core.NewRecord(collection)
		entry.Set("guild_id", guildID)
		entry.Set("user_id", userID)
		entry.Set("emoji_id", emojiID)
		entry.Set("emoji_name", emojiName)
		entry.Set("reactions", delta)
		if err := app.SaveWithContext(ctx, entry); err != nil {
			return err
		}
	}

	if eventBus != nil {
		if err := EnqueueLeaderboardUpdates(ctx, app, eventBus, logger, guildID, emojiID, emojiName); err != nil && logger != nil {
			logger.Warn("failed to enqueue leaderboard updates", slog.Any("err", err))
		}
	}

	return nil
}

func EnqueueLeaderboardUpdates(ctx context.Context, app core.App, eventBus *bus.Bus, logger *slog.Logger, guildID string, emojiID string, emojiName string) error {
	if app == nil || eventBus == nil {
		return nil
	}

	records, err := app.FindAllRecords("static_messages", dbx.HashExp{
		"guild_id": guildID,
		"type":     StaticMessageTypeLeaderboard,
		"update":   StaticMessageUpdateInstant,
	})
	if err != nil {
		return err
	}

	for _, record := range records {
		channelIDRaw := strings.TrimSpace(record.GetString("channel_id"))
		messageIDRaw := strings.TrimSpace(record.GetString("message_id"))
		if channelIDRaw == "" || messageIDRaw == "" {
			continue
		}
		channelID, err := snowflake.Parse(channelIDRaw)
		if err != nil {
			continue
		}
		messageID, err := snowflake.Parse(messageIDRaw)
		if err != nil {
			continue
		}

		config, err := ParseLeaderboardConfig(record.GetString("config"))
		if err != nil && logger != nil {
			logger.Warn("invalid leaderboard config", slog.Any("err", err))
		}

		if !emojiMatches(config.EmojiID, config.EmojiName, emojiID, emojiName) {
			continue
		}

		embed, err := buildLeaderboardEmbed(ctx, app, guildID, config)
		if err != nil {
			if logger != nil {
				logger.Warn("failed to build leaderboard embed", slog.Any("err", err))
			}
			continue
		}

		eventBus.DiscordActions <- bus.EditMessage{
			ChannelID: channelID,
			MessageID: messageID,
			Embeds:    []discord.Embed{embed},
		}
	}

	return nil
}

func ParseLeaderboardConfig(raw string) (leaderboardConfig, error) {
	config := leaderboardConfig{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return config, nil
	}
	err := json.Unmarshal([]byte(raw), &config)
	if err != nil {
		return config, err
	}
	return config, nil
}

func emojiMatches(configID string, configName string, eventID string, eventName string) bool {
	configID = strings.TrimSpace(configID)
	configName = strings.TrimSpace(configName)
	if configID != "" {
		return configID == strings.TrimSpace(eventID)
	}
	if configName != "" {
		return configName == strings.TrimSpace(eventName)
	}
	return false
}

func buildLeaderboardEmbed(ctx context.Context, app core.App, guildID string, config leaderboardConfig) (discord.Embed, error) {
	if app == nil {
		return discord.Embed{}, nil
	}

	emojiID := strings.TrimSpace(config.EmojiID)
	emojiName := strings.TrimSpace(config.EmojiName)
	if emojiID == "" && emojiName == "" {
		return discord.Embed{}, fmt.Errorf("missing emoji in config")
	}

	top := config.Top
	if top <= 0 {
		top = 10
	}

	title := strings.TrimSpace(config.Title)
	if title == "" {
		if track, _ := findReactionTrack(ctx, app, guildID, emojiID, emojiName); track != nil {
			title = strings.TrimSpace(track.Title)
		}
	}
	if title == "" {
		title = "Reaction Leaderboard"
	}

	display := emojiDisplay(emojiID, emojiName)
	if display != "" {
		title = fmt.Sprintf("%s %s", display, title)
	}

	entries, err := loadLeaderboardEntries(ctx, app, guildID, emojiID, emojiName)
	if err != nil {
		return discord.Embed{}, err
	}

	description := buildLeaderboardDescription(entries, top)
	if description == "" {
		description = "No reactions tracked yet."
	}

	return discordembed.BuildEmbed(discordembed.EmbedTemplate{
		Tone:        discordembed.EmbedInfo,
		Title:       title,
		Description: description,
	}), nil
}

func loadLeaderboardEntries(_ context.Context, app core.App, guildID string, emojiID string, emojiName string) ([]leaderboardEntry, error) {
	filter := dbx.HashExp{
		"guild_id": guildID,
	}
	if emojiID != "" {
		filter["emoji_id"] = emojiID
	} else {
		filter["emoji_name"] = emojiName
	}

	records, err := app.FindAllRecords("reaction_leaderboard", filter)
	if err != nil {
		return nil, err
	}

	entries := make([]leaderboardEntry, 0, len(records))
	for _, record := range records {
		entries = append(entries, leaderboardEntry{
			UserID:    record.GetString("user_id"),
			Reactions: record.GetInt("reactions"),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Reactions == entries[j].Reactions {
			return entries[i].UserID < entries[j].UserID
		}
		return entries[i].Reactions > entries[j].Reactions
	})

	return entries, nil
}

func buildLeaderboardDescription(entries []leaderboardEntry, top int) string {
	if len(entries) == 0 || top <= 0 {
		return ""
	}
	if len(entries) < top {
		top = len(entries)
	}

	width := len(fmt.Sprintf("%d", top))
	lines := make([]string, 0, top)
	for i := 0; i < top; i++ {
		entry := entries[i]
		if entry.UserID == "" {
			continue
		}
		rank := fmt.Sprintf("%0*d", width, i+1)
		lines = append(lines, fmt.Sprintf("%s. <@%s> — %d", rank, entry.UserID, entry.Reactions))
	}

	return strings.Join(lines, "\n")
}

func emojiDisplay(emojiID string, emojiName string) string {
	emojiID = strings.TrimSpace(emojiID)
	emojiName = strings.TrimSpace(emojiName)
	if emojiID != "" && emojiName != "" {
		return fmt.Sprintf("<:%s:%s>", emojiName, emojiID)
	}
	return emojiName
}

func findReactionTrack(_ context.Context, app core.App, guildID string, emojiID string, emojiName string) (*bus.ReactionTrack, error) {
	if app == nil {
		return nil, nil
	}

	filter := dbx.HashExp{
		"guild_id": guildID,
	}
	if emojiID != "" {
		filter["emoji_id"] = emojiID
	} else if emojiName != "" {
		filter["emoji_name"] = emojiName
	} else {
		return nil, nil
	}

	records, err := app.FindAllRecords("reaction_tracks", filter)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	record := records[0]
	return &bus.ReactionTrack{
		EmojiID:     strings.TrimSpace(record.GetString("emoji_id")),
		EmojiName:   strings.TrimSpace(record.GetString("emoji_name")),
		Title:       strings.TrimSpace(record.GetString("title")),
		Description: strings.TrimSpace(record.GetString("description")),
	}, nil
}
