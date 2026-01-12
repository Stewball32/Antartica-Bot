package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"antartica-bot/internal/bus"
	"antartica-bot/internal/discord/commands"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)

func (h *Handler) handleRoleToggleSelf(event *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) {
	if event.GuildID() == nil {
		_ = respondEphemeralTone(event, EmbedDecline, "This command can only be used in a server.")
		return
	}

	member := event.Member()
	if member == nil {
		_ = respondEphemeralTone(event, EmbedError, "Missing member data.")
		return
	}

	if h.roleToggleStore == nil {
		_ = respondEphemeralTone(event, EmbedError, "Role toggle store is not configured.")
		return
	}

	guildID := *event.GuildID()
	toggles, err := h.roleToggleStore.ListRoleToggles(context.Background(), guildID)
	if err != nil {
		h.logger.Error("failed to load role toggles", slog.Any("err", err))
		_ = respondEphemeralTone(event, EmbedError, "Failed to load role toggles.")
		return
	}
	if len(toggles) == 0 {
		_ = respondEphemeralTone(event, EmbedInfo, "No self-assignable roles are configured.")
		return
	}

	caches := event.Client().Caches()
	memberPermissions := member.Permissions
	if memberPermissions == 0 {
		memberPermissions = caches.MemberPermissions(member.Member)
	}

	botState, err := h.resolveBotRoleState(event, guildID)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("unable to resolve bot role state", slog.Any("err", err))
		}
		_ = respondEphemeralTone(event, EmbedWarn, "Bot permissions could not be verified yet.")
		return
	}

	if role, ok := data.OptRole("role"); ok {
		h.handleRoleToggleSelfRole(event, member, memberPermissions, toggles, role, botState)
		return
	}

	h.handleRoleToggleSelfList(event, member, memberPermissions, toggles, botState)
}

func (h *Handler) handleRoleToggleSelfRole(event *events.ApplicationCommandInteractionCreate, member *discord.ResolvedMember, memberPermissions discord.Permissions, toggles []bus.RoleToggle, role discord.Role, botState botRoleState) {
	toggle, ok := findRoleToggle(toggles, role.ID)
	if !ok {
		_ = respondEphemeralTone(event, EmbedDecline, "That role is not self-assignable.")
		return
	}

	requiredPermissions, err := parseRoleTogglePermissions(toggle.Permissions)
	if err != nil {
		if h.logger != nil {
			guildID := *event.GuildID()
			h.logger.Warn("invalid role toggle permissions", slog.String("guild_id", guildID.String()), slog.String("role_id", role.ID.String()), slog.Any("err", err))
		}
		_ = respondEphemeralTone(event, EmbedError, "That role is misconfigured. Ask an admin to update it.")
		return
	}
	if requiredPermissions != 0 && !memberPermissions.Has(requiredPermissions) {
		_ = respondEphemeralTone(event, EmbedDecline, "You don't have permission to toggle that role.")
		return
	}

	if ok, reason := canBotManageRoleWithState(event, role, botState); !ok {
		_ = respondEphemeralTone(event, EmbedDecline, reason)
		return
	}

	hasRole := memberHasRole(member.RoleIDs, role.ID)
	updatedRoles := append([]snowflake.ID(nil), member.RoleIDs...)
	if hasRole {
		updatedRoles, _ = removeRole(updatedRoles, role.ID)
	} else {
		updatedRoles = addRole(updatedRoles, role.ID)
	}

	_, err = event.Client().Rest().UpdateMember(*event.GuildID(), member.User.ID, discord.MemberUpdate{Roles: &updatedRoles})
	if err != nil {
		if h.logger != nil {
			h.logger.Error("failed to toggle role", slog.Any("err", err))
		}
		_ = respondEphemeralTone(event, EmbedError, "Failed to toggle the role.")
		return
	}

	if hasRole {
		_ = respondEphemeralTone(event, EmbedSuccess, fmt.Sprintf("Removed %s.", role.Mention()))
		return
	}
	_ = respondEphemeralTone(event, EmbedSuccess, fmt.Sprintf("Added %s.", role.Mention()))
}

func (h *Handler) handleRoleToggleSelfList(event *events.ApplicationCommandInteractionCreate, member *discord.ResolvedMember, memberPermissions discord.Permissions, toggles []bus.RoleToggle, botState botRoleState) {
	guildID := *event.GuildID()
	caches := event.Client().Caches()
	roleMap := make(map[snowflake.ID]discord.Role, len(toggles))
	missingRoles := false
	for _, toggle := range toggles {
		if role, ok := caches.Role(guildID, toggle.RoleID); ok {
			roleMap[toggle.RoleID] = role
		} else {
			missingRoles = true
		}
	}
	if missingRoles {
		roles, err := event.Client().Rest().GetRoles(guildID)
		if err != nil {
			if h.logger != nil {
				h.logger.Warn("failed to fetch guild roles", slog.Any("err", err))
			}
		} else {
			for _, role := range roles {
				roleMap[role.ID] = role
			}
		}
	}

	haveLines := make([]string, 0)
	availableLines := make([]string, 0)

	for _, toggle := range toggles {
		role, ok := roleMap[toggle.RoleID]

		requiredPermissions, err := parseRoleTogglePermissions(toggle.Permissions)
		if err != nil {
			if h.logger != nil {
				h.logger.Warn("invalid role toggle permissions", slog.String("guild_id", guildID.String()), slog.String("role_id", toggle.RoleID.String()), slog.Any("err", err))
			}
			continue
		}
		if requiredPermissions != 0 && !memberPermissions.Has(requiredPermissions) {
			continue
		}

		if ok {
			if ok, _ := canBotManageRoleWithState(event, role, botState); !ok {
				continue
			}
		}

		roleMention := fmt.Sprintf("<@&%s>", toggle.RoleID.String())
		if ok {
			roleMention = role.Mention()
		}
		line := formatRoleLineMention(roleMention, toggle.Description)
		roleID := toggle.RoleID
		if ok {
			roleID = role.ID
		}
		if memberHasRole(member.RoleIDs, roleID) {
			haveLines = append(haveLines, line)
		} else {
			availableLines = append(availableLines, line)
		}
	}

	if len(haveLines) == 0 && len(availableLines) == 0 {
		_ = respondEphemeralTone(event, EmbedInfo, "No self-assignable roles are available to you.")
		return
	}

	haveValue := "None"
	if len(haveLines) > 0 {
		haveValue = strings.Join(haveLines, "\n")
	}
	availableValue := "None"
	if len(availableLines) > 0 {
		availableValue = strings.Join(availableLines, "\n")
	}

	embed := BuildEmbed(EmbedTemplate{
		Tone:        EmbedInfo,
		Title:       "Self-assignable roles",
		Description: fmt.Sprintf("Use `/%s` with a role to toggle it.", commands.RoleSelfCommandName),
		Fields: []discord.EmbedField{
			{Name: "You have", Value: haveValue},
			{Name: "Available", Value: availableValue},
		},
	})

	_ = event.CreateMessage(discord.MessageCreate{
		Embeds: []discord.Embed{embed},
		Flags:  discord.MessageFlagEphemeral,
	})
}

func findRoleToggle(toggles []bus.RoleToggle, roleID snowflake.ID) (bus.RoleToggle, bool) {
	for _, toggle := range toggles {
		if toggle.RoleID == roleID {
			return toggle, true
		}
	}
	return bus.RoleToggle{}, false
}

func parseRoleTogglePermissions(value string) (discord.Permissions, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, err
	}
	if parsed < 0 {
		return 0, fmt.Errorf("permissions must be positive")
	}
	return discord.Permissions(parsed), nil
}

func formatRoleLineMention(mention string, description string) string {
	description = strings.TrimSpace(description)
	if description == "" {
		return mention
	}
	return fmt.Sprintf("%s - %s", mention, description)
}

func memberHasRole(roleIDs []snowflake.ID, roleID snowflake.ID) bool {
	for _, id := range roleIDs {
		if id == roleID {
			return true
		}
	}
	return false
}

func addRole(roleIDs []snowflake.ID, roleID snowflake.ID) []snowflake.ID {
	if memberHasRole(roleIDs, roleID) {
		return roleIDs
	}
	return append(roleIDs, roleID)
}

func removeRole(roleIDs []snowflake.ID, roleID snowflake.ID) ([]snowflake.ID, bool) {
	updated := make([]snowflake.ID, 0, len(roleIDs))
	removed := false
	for _, id := range roleIDs {
		if id == roleID {
			removed = true
			continue
		}
		updated = append(updated, id)
	}
	return updated, removed
}
