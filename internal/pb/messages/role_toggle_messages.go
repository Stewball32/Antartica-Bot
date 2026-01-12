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
	"antartica-bot/internal/discord/commands"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const StaticMessageTypeRoleToggles = "role_toggles"

type roleToggleMessageConfig struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

func EnqueueRoleToggleUpdates(ctx context.Context, app core.App, eventBus *bus.Bus, logger *slog.Logger, guildID string) error {
	if app == nil || eventBus == nil {
		return nil
	}

	records, err := app.FindAllRecords("static_messages", dbx.HashExp{
		"guild_id": guildID,
		"type":     StaticMessageTypeRoleToggles,
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

		config, err := parseRoleToggleMessageConfig(record.GetString("config"))
		if err != nil && logger != nil {
			logger.Warn("invalid role toggle message config", slog.Any("err", err))
		}

		embed, err := buildRoleToggleEmbed(ctx, app, guildID, config)
		if err != nil {
			if logger != nil {
				logger.Warn("failed to build role toggle embed", slog.Any("err", err))
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

func buildRoleToggleEmbed(_ context.Context, app core.App, guildID string, config roleToggleMessageConfig) (discord.Embed, error) {
	if app == nil {
		return discord.Embed{}, nil
	}

	title := strings.TrimSpace(config.Title)
	if title == "" {
		title = "Self-assignable roles"
	}
	description := strings.TrimSpace(config.Description)
	if description == "" {
		description = fmt.Sprintf("Use `/%s` to add or remove roles.", commands.RoleSelfCommandName)
	}

	lines, err := loadRoleToggleLines(app, guildID)
	if err != nil {
		return discord.Embed{}, err
	}

	rolesValue := "No self-assignable roles are configured."
	if len(lines) > 0 {
		rolesValue = strings.Join(lines, "\n")
	}

	return discordembed.BuildEmbed(discordembed.EmbedTemplate{
		Tone:        discordembed.EmbedInfo,
		Title:       title,
		Description: description,
		Fields: []discord.EmbedField{
			{Name: "Roles", Value: rolesValue},
		},
	}), nil
}

func loadRoleToggleLines(app core.App, guildID string) ([]string, error) {
	if app == nil {
		return nil, nil
	}

	records, err := app.FindAllRecords("role_toggles", dbx.HashExp{
		"guild_id": guildID,
	})
	if err != nil {
		return nil, err
	}

	type roleToggleEntry struct {
		RoleID      string
		Description string
	}

	entries := make([]roleToggleEntry, 0, len(records))
	for _, record := range records {
		roleID := strings.TrimSpace(record.GetString("role_id"))
		if roleID == "" {
			continue
		}
		entries = append(entries, roleToggleEntry{
			RoleID:      roleID,
			Description: strings.TrimSpace(record.GetString("description")),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].RoleID < entries[j].RoleID
	})

	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		line := fmt.Sprintf("<@&%s>", entry.RoleID)
		if entry.Description != "" {
			line = fmt.Sprintf("%s - %s", line, entry.Description)
		}
		lines = append(lines, line)
	}

	return lines, nil
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
