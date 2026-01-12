package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)

var customEmojiPattern = regexp.MustCompile(`^<a?:([a-zA-Z0-9_]+):([0-9]+)>$`)

type leaderboardMessageConfig struct {
	EmojiID   string `json:"emoji_id,omitempty"`
	EmojiName string `json:"emoji_name,omitempty"`
	Top       int    `json:"top,omitempty"`
	Title     string `json:"title,omitempty"`
}

const staticMessageTypeLeaderboard = "reaction_leaderboard"

func (h *Handler) handleReactionCommand(event *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) {
	if event.GuildID() == nil {
		_ = respondEphemeralTone(event, EmbedDecline, "This command can only be used in a server.")
		return
	}

	switch data.CommandPath() {
	case "/reaction/add":
		h.handleAdminReactionAdd(event, data)
	case "/reaction/remove":
		h.handleAdminReactionRemove(event, data)
	case "/reaction/list":
		h.handleAdminReactionList(event)
	case "/reaction/leaderboard/create":
		h.handleAdminLeaderboardCreate(event, data)
	case "/reaction/leaderboard/remove":
		h.handleAdminLeaderboardRemove(event, data)
	case "/reaction/leaderboard/list":
		h.handleAdminLeaderboardList(event)
	default:
		_ = respondEphemeralTone(event, EmbedWarn, "Unknown subcommand.")
	}
}

func (h *Handler) handleAdminReactionAdd(event *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) {
	if h.reactionTrackStore == nil {
		_ = respondEphemeralTone(event, EmbedError, "Reaction track store is not configured.")
		return
	}

	rawEmoji, ok := data.OptString("emoji")
	if !ok || strings.TrimSpace(rawEmoji) == "" {
		_ = respondEphemeralTone(event, EmbedWarn, "Emoji is required.")
		return
	}

	title, ok := data.OptString("title")
	title = strings.TrimSpace(title)
	if !ok || title == "" {
		_ = respondEphemeralTone(event, EmbedWarn, "Title is required.")
		return
	}

	emojiID, emojiName, err := parseEmojiInput(rawEmoji)
	if err != nil {
		_ = respondEphemeralTone(event, EmbedWarn, err.Error())
		return
	}

	if emojiID != "" {
		guildID := *event.GuildID()
		ok, err := ensureGuildEmoji(event, guildID, emojiID)
		if err != nil {
			_ = respondEphemeralTone(event, EmbedWarn, "Couldn't verify that emoji. Make sure it belongs to this server or use a unicode emoji.")
			return
		}
		if !ok {
			_ = respondEphemeralTone(event, EmbedDecline, "Custom emojis must belong to this server.")
			return
		}
	}

	description, _ := data.OptString("description")
	description = strings.TrimSpace(description)

	guildID := *event.GuildID()
	created, err := h.reactionTrackStore.UpsertReactionTrack(context.Background(), guildID, emojiID, emojiName, title, description)
	if err != nil {
		_ = respondEphemeralTone(event, EmbedError, "Failed to save tracked reaction.")
		return
	}

	display := formatEmojiDisplay(emojiID, emojiName)
	if created {
		_ = respondEphemeralTone(event, EmbedSuccess, fmt.Sprintf("Tracking %s for reactions.", display))
		return
	}
	_ = respondEphemeralTone(event, EmbedSuccess, fmt.Sprintf("Updated tracking for %s.", display))
}

func (h *Handler) handleAdminReactionRemove(event *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) {
	if h.reactionTrackStore == nil {
		_ = respondEphemeralTone(event, EmbedError, "Reaction track store is not configured.")
		return
	}

	rawEmoji, ok := data.OptString("emoji")
	if !ok || strings.TrimSpace(rawEmoji) == "" {
		_ = respondEphemeralTone(event, EmbedWarn, "Emoji is required.")
		return
	}

	emojiID, emojiName, err := parseEmojiInput(rawEmoji)
	if err != nil {
		_ = respondEphemeralTone(event, EmbedWarn, err.Error())
		return
	}

	guildID := *event.GuildID()
	deleted, err := h.reactionTrackStore.RemoveReactionTrack(context.Background(), guildID, emojiID, emojiName)
	if err != nil {
		_ = respondEphemeralTone(event, EmbedError, "Failed to remove tracked reaction.")
		return
	}

	display := formatEmojiDisplay(emojiID, emojiName)
	if deleted == 0 {
		_ = respondEphemeralTone(event, EmbedInfo, fmt.Sprintf("%s was not being tracked.", display))
		return
	}
	_ = respondEphemeralTone(event, EmbedSuccess, fmt.Sprintf("Stopped tracking %s.", display))
}

func (h *Handler) handleAdminReactionList(event *events.ApplicationCommandInteractionCreate) {
	if h.reactionTrackStore == nil {
		_ = respondEphemeralTone(event, EmbedError, "Reaction track store is not configured.")
		return
	}

	guildID := *event.GuildID()
	tracks, err := h.reactionTrackStore.ListReactionTracks(context.Background(), guildID)
	if err != nil {
		_ = respondEphemeralTone(event, EmbedError, "Failed to load tracked reactions.")
		return
	}
	if len(tracks) == 0 {
		_ = respondEphemeralTone(event, EmbedInfo, "No reactions are being tracked.")
		return
	}

	lines := make([]string, 0, len(tracks))
	for _, track := range tracks {
		display := formatEmojiDisplay(track.EmojiID, track.EmojiName)
		line := fmt.Sprintf("%s — %s", display, track.Title)
		if track.Description != "" {
			line = fmt.Sprintf("%s (%s)", line, track.Description)
		}
		lines = append(lines, line)
	}

	embed := BuildEmbed(EmbedTemplate{
		Tone:        EmbedInfo,
		Title:       "Tracked Reactions",
		Description: strings.Join(lines, "\n"),
	})
	_ = event.CreateMessage(discord.MessageCreate{
		Embeds: []discord.Embed{embed},
		Flags:  discord.MessageFlagEphemeral,
	})
}

func (h *Handler) handleAdminLeaderboardCreate(event *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) {
	if h.staticMessageStore == nil {
		_ = respondEphemeralTone(event, EmbedError, "Static message store is not configured.")
		return
	}

	channel, ok := data.OptChannel("channel")
	if !ok {
		_ = respondEphemeralTone(event, EmbedWarn, "Channel is required.")
		return
	}
	channelID := channel.ID

	rawEmoji, ok := data.OptString("emoji")
	if !ok || strings.TrimSpace(rawEmoji) == "" {
		_ = respondEphemeralTone(event, EmbedWarn, "Emoji is required.")
		return
	}
	emojiID, emojiName, err := parseEmojiInput(rawEmoji)
	if err != nil {
		_ = respondEphemeralTone(event, EmbedWarn, err.Error())
		return
	}

	update, ok := data.OptString("update")
	update = strings.TrimSpace(strings.ToLower(update))
	if !ok || update == "" {
		_ = respondEphemeralTone(event, EmbedWarn, "Update cadence is required.")
		return
	}
	if update != "instant" && update != "hourly" && update != "daily" {
		_ = respondEphemeralTone(event, EmbedWarn, "Update must be instant, hourly, or daily.")
		return
	}

	top := 0
	if value, ok := data.OptInt("top"); ok {
		top = value
	}

	title, _ := data.OptString("title")
	title = strings.TrimSpace(title)

	configBytes, err := json.Marshal(leaderboardMessageConfig{
		EmojiID:   emojiID,
		EmojiName: emojiName,
		Top:       top,
		Title:     title,
	})
	if err != nil {
		_ = respondEphemeralTone(event, EmbedError, "Failed to serialize leaderboard config.")
		return
	}

	embed := BuildEmbed(EmbedTemplate{
		Tone:        EmbedInfo,
		Title:       "Leaderboard",
		Description: "Setting up leaderboard...",
	})

	message, err := event.Client().Rest().CreateMessage(channelID, discord.MessageCreate{
		Embeds: []discord.Embed{embed},
	})
	if err != nil {
		_ = respondEphemeralTone(event, EmbedError, "Failed to create the leaderboard message.")
		return
	}

	guildID := *event.GuildID()
	if err := h.staticMessageStore.CreateStaticMessage(context.Background(), guildID, channelID, message.ID, staticMessageTypeLeaderboard, string(configBytes), update); err != nil {
		_ = event.Client().Rest().DeleteMessage(channelID, message.ID)
		_ = respondEphemeralTone(event, EmbedError, "Failed to store the leaderboard message.")
		return
	}

	_ = respondEphemeralTone(event, EmbedSuccess, fmt.Sprintf("Leaderboard message created in <#%s>.", channelID.String()))
}

func (h *Handler) handleAdminLeaderboardRemove(event *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) {
	if h.staticMessageStore == nil {
		_ = respondEphemeralTone(event, EmbedError, "Static message store is not configured.")
		return
	}

	messageRaw, ok := data.OptString("message_id")
	messageRaw = strings.TrimSpace(messageRaw)
	if !ok || messageRaw == "" {
		_ = respondEphemeralTone(event, EmbedWarn, "Message ID is required.")
		return
	}

	messageID, err := snowflake.Parse(messageRaw)
	if err != nil {
		_ = respondEphemeralTone(event, EmbedWarn, "Message ID must be a number.")
		return
	}

	guildID := *event.GuildID()
	deleted, err := h.staticMessageStore.RemoveStaticMessage(context.Background(), guildID, messageID)
	if err != nil {
		_ = respondEphemeralTone(event, EmbedError, "Failed to remove the leaderboard message.")
		return
	}
	if deleted == 0 {
		_ = respondEphemeralTone(event, EmbedInfo, "That message was not registered.")
		return
	}
	_ = respondEphemeralTone(event, EmbedSuccess, "Leaderboard message removed.")
}

func (h *Handler) handleAdminLeaderboardList(event *events.ApplicationCommandInteractionCreate) {
	if h.staticMessageStore == nil {
		_ = respondEphemeralTone(event, EmbedError, "Static message store is not configured.")
		return
	}

	guildID := *event.GuildID()
	messages, err := h.staticMessageStore.ListStaticMessages(context.Background(), guildID)
	if err != nil {
		_ = respondEphemeralTone(event, EmbedError, "Failed to load leaderboard messages.")
		return
	}
	if len(messages) == 0 {
		_ = respondEphemeralTone(event, EmbedInfo, "No leaderboard messages are configured.")
		return
	}

	lines := make([]string, 0, len(messages))
	for _, message := range messages {
		if message.Type != staticMessageTypeLeaderboard {
			continue
		}
		emojiDisplay := ""
		if config, err := parseLeaderboardConfig(message.Config); err == nil {
			emojiDisplay = formatEmojiDisplay(config.EmojiID, config.EmojiName)
		}
		if emojiDisplay == "" {
			emojiDisplay = "emoji"
		}
		line := fmt.Sprintf("%s — <#%s> — %s", emojiDisplay, message.ChannelID.String(), message.Update)
		lines = append(lines, line)
	}

	embed := BuildEmbed(EmbedTemplate{
		Tone:        EmbedInfo,
		Title:       "Leaderboard Messages",
		Description: strings.Join(lines, "\n"),
	})

	_ = event.CreateMessage(discord.MessageCreate{
		Embeds: []discord.Embed{embed},
		Flags:  discord.MessageFlagEphemeral,
	})
}

func parseEmojiInput(raw string) (string, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", fmt.Errorf("emoji is required")
	}

	if matches := customEmojiPattern.FindStringSubmatch(raw); len(matches) == 3 {
		return matches[2], matches[1], nil
	}

	return "", raw, nil
}

func formatEmojiDisplay(emojiID string, emojiName string) string {
	emojiID = strings.TrimSpace(emojiID)
	emojiName = strings.TrimSpace(emojiName)
	if emojiID != "" && emojiName != "" {
		return fmt.Sprintf("<:%s:%s>", emojiName, emojiID)
	}
	if emojiName != "" {
		return emojiName
	}
	if emojiID != "" {
		return fmt.Sprintf("<:%s:%s>", "emoji", emojiID)
	}
	return "emoji"
}

func ensureGuildEmoji(event *events.ApplicationCommandInteractionCreate, guildID snowflake.ID, emojiID string) (bool, error) {
	emojiID = strings.TrimSpace(emojiID)
	if emojiID == "" {
		return true, nil
	}

	parsed, err := snowflake.Parse(emojiID)
	if err != nil {
		return false, err
	}

	caches := event.Client().Caches()
	if _, ok := caches.Emoji(guildID, parsed); ok {
		return true, nil
	}

	emoji, err := event.Client().Rest().GetEmoji(guildID, parsed)
	if err != nil {
		return false, err
	}
	return emoji != nil, nil
}

func parseLeaderboardConfig(raw string) (leaderboardMessageConfig, error) {
	config := leaderboardMessageConfig{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return config, nil
	}
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return config, err
	}
	return config, nil
}
