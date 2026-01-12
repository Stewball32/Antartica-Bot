package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)

const staticMessageTypeRoleToggles = "role_toggles"

type roleToggleMessageConfig struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

func (h *Handler) handleRoleToggleMessageCreate(event *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) {
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

	title, _ := data.OptString("title")
	title = strings.TrimSpace(title)
	description, _ := data.OptString("description")
	description = strings.TrimSpace(description)

	config := ""
	if title != "" || description != "" {
		configBytes, err := json.Marshal(roleToggleMessageConfig{
			Title:       title,
			Description: description,
		})
		if err != nil {
			_ = respondEphemeralTone(event, EmbedError, "Failed to serialize message config.")
			return
		}
		config = string(configBytes)
	}

	embed := BuildEmbed(EmbedTemplate{
		Tone:        EmbedInfo,
		Title:       "Self-assignable roles",
		Description: "Setting up the role list...",
	})

	message, err := event.Client().Rest().CreateMessage(channelID, discord.MessageCreate{
		Embeds: []discord.Embed{embed},
	})
	if err != nil {
		_ = respondEphemeralTone(event, EmbedError, "Failed to create the role list message.")
		return
	}

	guildID := *event.GuildID()
	if err := h.staticMessageStore.CreateStaticMessage(context.Background(), guildID, channelID, message.ID, staticMessageTypeRoleToggles, config, "instant"); err != nil {
		_ = event.Client().Rest().DeleteMessage(channelID, message.ID)
		_ = respondEphemeralTone(event, EmbedError, "Failed to store the role list message.")
		return
	}

	_ = respondEphemeralTone(event, EmbedSuccess, fmt.Sprintf("Role list message created in <#%s>.", channelID.String()))
}

func (h *Handler) handleRoleToggleMessageRemove(event *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) {
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
		_ = respondEphemeralTone(event, EmbedError, "Failed to remove the role list message.")
		return
	}
	if deleted == 0 {
		_ = respondEphemeralTone(event, EmbedInfo, "That message was not registered.")
		return
	}
	_ = respondEphemeralTone(event, EmbedSuccess, "Role list message removed.")
}

func (h *Handler) handleRoleToggleMessageList(event *events.ApplicationCommandInteractionCreate) {
	if h.staticMessageStore == nil {
		_ = respondEphemeralTone(event, EmbedError, "Static message store is not configured.")
		return
	}

	guildID := *event.GuildID()
	messages, err := h.staticMessageStore.ListStaticMessages(context.Background(), guildID)
	if err != nil {
		_ = respondEphemeralTone(event, EmbedError, "Failed to load role list messages.")
		return
	}

	lines := make([]string, 0)
	for _, message := range messages {
		if message.Type != staticMessageTypeRoleToggles {
			continue
		}
		title := ""
		if config, err := parseRoleToggleMessageConfig(message.Config); err == nil {
			title = strings.TrimSpace(config.Title)
		}
		line := fmt.Sprintf("<#%s> - %s", message.ChannelID.String(), message.MessageID.String())
		if title != "" {
			line = fmt.Sprintf("%s - %s", line, title)
		}
		lines = append(lines, line)
	}

	if len(lines) == 0 {
		_ = respondEphemeralTone(event, EmbedInfo, "No role list messages are configured.")
		return
	}

	embed := BuildEmbed(EmbedTemplate{
		Tone:        EmbedInfo,
		Title:       "Role List Messages",
		Description: strings.Join(lines, "\n"),
	})

	_ = event.CreateMessage(discord.MessageCreate{
		Embeds: []discord.Embed{embed},
		Flags:  discord.MessageFlagEphemeral,
	})
}

func parseRoleToggleMessageConfig(raw string) (roleToggleMessageConfig, error) {
	config := roleToggleMessageConfig{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return config, nil
	}
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return config, err
	}
	return config, nil
}
