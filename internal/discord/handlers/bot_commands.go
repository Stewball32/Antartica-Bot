package handlers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
)

func (h *Handler) handleBotCommand(event *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) {
	if event.GuildID() == nil {
		_ = respondEphemeralTone(event, EmbedDecline, "This command can only be used in a server.")
		return
	}

	switch data.CommandPath() {
	case "/bot/name":
		h.handleBotName(event, data)
	case "/bot/avatar":
		h.handleBotAvatar(event, data)
	case "/bot/banner":
		h.handleBotBanner(event, data)
	case "/bot/about":
		h.handleBotAbout(event, data)
	case "/bot/status":
		h.handleBotStatus(event, data)
	case "/bot/activity":
		h.handleBotActivity(event, data)
	default:
		_ = respondEphemeralTone(event, EmbedWarn, "Unknown subcommand.")
	}
}

func (h *Handler) handleBotName(event *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) {
	name, ok := data.OptString("name")
	name = strings.TrimSpace(name)
	if !ok || name == "" {
		_ = respondEphemeralTone(event, EmbedWarn, "Name is required.")
		return
	}

	_, err := event.Client().Rest().UpdateCurrentUser(discord.UserUpdate{Username: name})
	if err != nil {
		h.logger.Error("failed to update bot name", slog.Any("err", err))
		_ = respondEphemeralTone(event, EmbedError, "Failed to update bot name.")
		return
	}

	_ = respondEphemeralTone(event, EmbedSuccess, "Bot name updated.")
}

func (h *Handler) handleBotAvatar(event *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) {
	icon, err := resolveBotIcon(event, data)
	if err != nil {
		_ = respondEphemeralTone(event, EmbedWarn, err.Error())
		return
	}

	_, err = event.Client().Rest().UpdateCurrentUser(discord.UserUpdate{
		Avatar: json.NewNullablePtr(*icon),
	})
	if err != nil {
		h.logger.Error("failed to update bot avatar", slog.Any("err", err))
		_ = respondEphemeralTone(event, EmbedError, "Failed to update bot avatar.")
		return
	}

	_ = respondEphemeralTone(event, EmbedSuccess, "Bot avatar updated.")
}

func (h *Handler) handleBotBanner(event *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) {
	icon, err := resolveBotIcon(event, data)
	if err != nil {
		_ = respondEphemeralTone(event, EmbedWarn, err.Error())
		return
	}

	_, err = event.Client().Rest().UpdateCurrentUser(discord.UserUpdate{
		Banner: json.NewNullablePtr(*icon),
	})
	if err != nil {
		h.logger.Error("failed to update bot banner", slog.Any("err", err))
		_ = respondEphemeralTone(event, EmbedError, "Failed to update bot banner.")
		return
	}

	_ = respondEphemeralTone(event, EmbedSuccess, "Bot banner updated.")
}

func (h *Handler) handleBotAbout(event *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) {
	about, ok := data.OptString("text")
	about = strings.TrimSpace(about)
	if !ok || about == "" {
		_ = respondEphemeralTone(event, EmbedWarn, "About text is required.")
		return
	}

	_, err := event.Client().Rest().UpdateCurrentApplication(discord.ApplicationUpdate{
		Description: &about,
	})
	if err != nil {
		h.logger.Error("failed to update bot description", slog.Any("err", err))
		_ = respondEphemeralTone(event, EmbedError, "Failed to update bot description.")
		return
	}

	_ = respondEphemeralTone(event, EmbedSuccess, "Bot description updated.")
}

func (h *Handler) handleBotStatus(event *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) {
	raw, _ := data.OptString("status")
	raw = strings.TrimSpace(strings.ToLower(raw))
	status := discord.OnlineStatusOnline
	if raw != "" {
		parsed, ok := parseOnlineStatus(raw)
		if !ok {
			_ = respondEphemeralTone(event, EmbedWarn, "Status must be online, idle, dnd, invisible, or offline.")
			return
		}
		status = parsed
	}

	if err := event.Client().SetPresence(context.Background(), gateway.WithOnlineStatus(status)); err != nil {
		h.logger.Error("failed to update bot status", slog.Any("err", err))
		_ = respondEphemeralTone(event, EmbedError, "Failed to update bot status.")
		return
	}

	_ = respondEphemeralTone(event, EmbedSuccess, "Bot status updated.")
}

func (h *Handler) handleBotActivity(event *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) {
	typeRaw, _ := data.OptString("type")
	typeRaw = strings.TrimSpace(strings.ToLower(typeRaw))
	text, _ := data.OptString("text")
	text = strings.TrimSpace(text)
	emojiRaw, _ := data.OptString("emoji")
	emojiRaw = strings.TrimSpace(emojiRaw)

	if text == "" {
		if err := event.Client().SetPresence(context.Background(), clearActivityPresence()); err != nil {
			h.logger.Error("failed to clear bot activity", slog.Any("err", err))
			_ = respondEphemeralTone(event, EmbedError, "Failed to clear bot activity.")
			return
		}
		_ = respondEphemeralTone(event, EmbedSuccess, "Bot activity cleared.")
		return
	}

	if typeRaw == "" {
		_ = respondEphemeralTone(event, EmbedWarn, "Activity type is required when setting activity.")
		return
	}

	if emojiRaw != "" && typeRaw != "custom" {
		_ = respondEphemeralTone(event, EmbedWarn, "Emoji is only supported for custom activity.")
		return
	}

	var opt gateway.PresenceOpt
	switch typeRaw {
	case "playing":
		opt = gateway.WithPlayingActivity(text)
	case "listening":
		opt = gateway.WithListeningActivity(text)
	case "watching":
		opt = gateway.WithWatchingActivity(text)
	case "competing":
		opt = gateway.WithCompetingActivity(text)
	case "custom":
		activity := discord.Activity{
			Name:  "Custom Status",
			Type:  discord.ActivityTypeCustom,
			State: &text,
		}
		if emojiRaw != "" {
			emoji, err := parseActivityEmoji(emojiRaw)
			if err != nil {
				_ = respondEphemeralTone(event, EmbedWarn, err.Error())
				return
			}
			activity.Emoji = emoji
		}
		opt = presenceWithActivity(activity)
	default:
		_ = respondEphemeralTone(event, EmbedWarn, "Activity type must be playing, listening, watching, competing, or custom.")
		return
	}

	if err := event.Client().SetPresence(context.Background(), opt); err != nil {
		h.logger.Error("failed to update bot activity", slog.Any("err", err))
		_ = respondEphemeralTone(event, EmbedError, "Failed to update bot activity.")
		return
	}

	_ = respondEphemeralTone(event, EmbedSuccess, "Bot activity updated.")
}

func resolveBotIcon(event *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) (*discord.Icon, error) {
	attachment, hasAttachment := data.OptAttachment("attachment")
	urlRaw, _ := data.OptString("url")
	urlRaw = strings.TrimSpace(urlRaw)

	if hasAttachment && urlRaw != "" {
		return nil, fmt.Errorf("Provide either an attachment or a URL, not both.")
	}
	if !hasAttachment && urlRaw == "" {
		return nil, fmt.Errorf("Provide an attachment or a URL.")
	}

	imageURL := urlRaw
	contentType := ""
	if hasAttachment {
		imageURL = attachment.URL
		if attachment.ContentType != nil {
			contentType = *attachment.ContentType
		}
	}

	return fetchIconFromURL(imageURL, contentType)
}

func fetchIconFromURL(imageURL string, contentTypeHint string) (*discord.Icon, error) {
	parsed, err := url.Parse(imageURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("Image URL must be a valid http(s) URL.")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("Image URL must start with http or https.")
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch image.")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch image.")
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Image URL returned status %d.", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil || len(data) == 0 {
		return nil, fmt.Errorf("Failed to read image data.")
	}

	iconType, ok := detectIconType(contentTypeHint, resp.Header.Get("Content-Type"), imageURL, data)
	if !ok {
		return nil, fmt.Errorf("Image must be PNG, JPEG, WEBP, or GIF.")
	}

	icon := discord.NewIconRaw(iconType, data)
	return icon, nil
}

func detectIconType(contentTypeHint string, headerContentType string, imageURL string, data []byte) (discord.IconType, bool) {
	if iconType, ok := iconTypeFromContentType(contentTypeHint); ok {
		return iconType, true
	}
	if iconType, ok := iconTypeFromContentType(headerContentType); ok {
		return iconType, true
	}
	if len(data) > 0 {
		if iconType, ok := iconTypeFromContentType(http.DetectContentType(data)); ok {
			return iconType, true
		}
	}
	if imageURL != "" {
		switch strings.ToLower(path.Ext(imageURL)) {
		case ".png":
			return discord.IconTypePNG, true
		case ".jpg", ".jpeg":
			return discord.IconTypeJPEG, true
		case ".webp":
			return discord.IconTypeWEBP, true
		case ".gif":
			return discord.IconTypeGIF, true
		}
	}
	return "", false
}

func iconTypeFromContentType(contentType string) (discord.IconType, bool) {
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		return "", false
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err == nil && mediaType != "" {
		contentType = mediaType
	}
	switch strings.ToLower(contentType) {
	case "image/png":
		return discord.IconTypePNG, true
	case "image/jpeg", "image/jpg":
		return discord.IconTypeJPEG, true
	case "image/webp":
		return discord.IconTypeWEBP, true
	case "image/gif":
		return discord.IconTypeGIF, true
	default:
		return "", false
	}
}

func parseOnlineStatus(raw string) (discord.OnlineStatus, bool) {
	switch raw {
	case "online":
		return discord.OnlineStatusOnline, true
	case "idle":
		return discord.OnlineStatusIdle, true
	case "dnd":
		return discord.OnlineStatusDND, true
	case "invisible":
		return discord.OnlineStatusInvisible, true
	case "offline":
		return discord.OnlineStatusInvisible, true
	default:
		return "", false
	}
}

func clearActivityPresence() gateway.PresenceOpt {
	return func(presence *gateway.MessageDataPresenceUpdate) {
		presence.Activities = []discord.Activity{}
	}
}

func presenceWithActivity(activity discord.Activity) gateway.PresenceOpt {
	return func(presence *gateway.MessageDataPresenceUpdate) {
		presence.Activities = []discord.Activity{activity}
	}
}

func parseActivityEmoji(raw string) (*discord.PartialEmoji, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("Emoji is required.")
	}

	emojiID, emojiName, err := parseEmojiInput(raw)
	if err != nil {
		return nil, fmt.Errorf("Emoji is invalid.")
	}

	emoji := &discord.PartialEmoji{}
	if emojiName != "" {
		emoji.Name = &emojiName
	}
	if emojiID != "" {
		parsed, err := snowflake.Parse(emojiID)
		if err != nil {
			return nil, fmt.Errorf("Emoji is invalid.")
		}
		emoji.ID = &parsed
	}
	if strings.HasPrefix(raw, "<a:") {
		emoji.Animated = true
	}

	if emoji.Name == nil && emoji.ID == nil {
		return nil, fmt.Errorf("Emoji is invalid.")
	}

	return emoji, nil
}
