package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"antartica-bot/internal/discord/commands"

	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)

func (h *Handler) OnApplicationCommand(event *events.ApplicationCommandInteractionCreate) {
	if event.ApplicationCommandInteraction.Data.Type() != discord.ApplicationCommandTypeSlash {
		return
	}

	data := event.SlashCommandInteractionData()
	switch data.CommandName() {
	case commands.RoleToggleCommandName:
		if event.GuildID() == nil {
			_ = respondEphemeralTone(event, EmbedDecline, "This command can only be used in a server.")
			return
		}

		switch data.CommandPath() {
		case "/role/add":
			h.handleRoleToggleAdd(event, data)
		case "/role/remove":
			h.handleRoleToggleRemove(event, data)
		case "/role/message-create":
			h.handleRoleToggleMessageCreate(event, data)
		case "/role/message-remove":
			h.handleRoleToggleMessageRemove(event, data)
		case "/role/message-list":
			h.handleRoleToggleMessageList(event)
		default:
			_ = respondEphemeralTone(event, EmbedWarn, "Unknown subcommand.")
		}
	case commands.RoleSelfCommandName:
		h.handleRoleToggleSelf(event, data)
	case commands.ReactionCommandName:
		h.handleReactionCommand(event, data)
	case commands.BotCommandName:
		h.handleBotCommand(event, data)
	}
}

func (h *Handler) handleRoleToggleAdd(event *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) {
	role, ok := data.OptRole("role")
	if !ok {
		_ = respondEphemeralTone(event, EmbedWarn, "Role is required.")
		return
	}

	descriptionRaw, ok := data.OptString("description")
	descriptionRaw = strings.TrimSpace(descriptionRaw)
	var description *string
	if ok && descriptionRaw != "" {
		description = &descriptionRaw
	}

	permissionsRaw, _ := data.OptString("permissions")
	permissionsRaw = strings.TrimSpace(permissionsRaw)
	if permissionsRaw == "" {
		permissionsRaw = "0"
	}
	permissionsValue, err := strconv.ParseUint(permissionsRaw, 10, 64)
	if err != nil {
		_ = respondEphemeralTone(event, EmbedWarn, "Permissions must be a whole number.")
		return
	}
	permissions := strconv.FormatUint(permissionsValue, 10)

	if ok, reason := h.canManageRole(event, role); !ok {
		_ = respondEphemeralTone(event, EmbedDecline, reason)
		return
	}

	if h.roleToggleStore == nil {
		_ = respondEphemeralTone(event, EmbedError, "Role toggle store is not configured.")
		return
	}

	guildID := *event.GuildID()
	created, err := h.roleToggleStore.UpsertRoleToggle(context.Background(), guildID, role.ID, permissions, description)
	if err != nil {
		h.logger.Error("failed to save role toggle", slog.Any("err", err))
		_ = respondEphemeralTone(event, EmbedError, "Failed to save role toggle.")
		return
	}

	if created {
		_ = respondEphemeralTone(event, EmbedSuccess, fmt.Sprintf("Added %s to the role toggles.", role.Mention()))
		return
	}
	_ = respondEphemeralTone(event, EmbedSuccess, fmt.Sprintf("Updated %s in the role toggles.", role.Mention()))
}

func (h *Handler) handleRoleToggleRemove(event *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) {
	role, ok := data.OptRole("role")
	if !ok {
		_ = respondEphemeralTone(event, EmbedWarn, "Role is required.")
		return
	}

	if ok, reason := h.canManageRole(event, role); !ok {
		_ = respondEphemeralTone(event, EmbedDecline, reason)
		return
	}

	if h.roleToggleStore == nil {
		_ = respondEphemeralTone(event, EmbedError, "Role toggle store is not configured.")
		return
	}

	guildID := *event.GuildID()
	deleted, err := h.roleToggleStore.RemoveRoleToggle(context.Background(), guildID, role.ID)
	if err != nil {
		h.logger.Error("failed to remove role toggle", slog.Any("err", err))
		_ = respondEphemeralTone(event, EmbedError, "Failed to remove role toggle.")
		return
	}

	if deleted == 0 {
		_ = respondEphemeralTone(event, EmbedInfo, fmt.Sprintf("%s was not in the role toggles.", role.Mention()))
		return
	}

	_ = respondEphemeralTone(event, EmbedSuccess, fmt.Sprintf("Removed %s from the role toggles.", role.Mention()))
}

func (h *Handler) canBotManageRole(event *events.ApplicationCommandInteractionCreate, role discord.Role) (bool, string) {
	guildID := *event.GuildID()

	if role.ID == guildID {
		return false, "You can't toggle the @everyone role."
	}
	if role.Managed {
		return false, "You can't toggle managed roles."
	}

	state, err := h.resolveBotRoleState(event, guildID)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("unable to resolve bot role state", slog.Any("err", err))
		}
		return false, "Bot permissions could not be verified yet."
	}
	return canBotManageRoleWithState(event, role, state)
}

func (h *Handler) canManageRole(event *events.ApplicationCommandInteractionCreate, role discord.Role) (bool, string) {
	guildID := *event.GuildID()

	member := event.Member()
	if member == nil {
		return false, "Missing member data."
	}

	caches := event.Client().Caches()
	memberPermissions := member.Permissions
	if memberPermissions == 0 {
		memberPermissions = caches.MemberPermissions(member.Member)
	}

	if !memberPermissions.Has(discord.PermissionManageRoles) {
		return false, "You need the Manage Roles permission."
	}

	if guild, ok := caches.Guild(guildID); ok && guild.OwnerID != member.User.ID {
		userTop := highestRolePosition(caches, guildID, member.RoleIDs)
		if role.Position >= userTop {
			return false, "You can only manage roles below your highest role."
		}
	}

	return h.canBotManageRole(event, role)
}

func highestRolePosition(caches cache.Caches, guildID snowflake.ID, roleIDs []snowflake.ID) int {
	top := 0
	for _, roleID := range roleIDs {
		if role, ok := caches.Role(guildID, roleID); ok {
			if role.Position > top {
				top = role.Position
			}
		}
	}
	return top
}

type botRoleState struct {
	permissions discord.Permissions
	topPosition int
}

func canBotManageRoleWithState(event *events.ApplicationCommandInteractionCreate, role discord.Role, state botRoleState) (bool, string) {
	guildID := *event.GuildID()
	if role.ID == guildID {
		return false, "You can't toggle the @everyone role."
	}
	if role.Managed {
		return false, "You can't toggle managed roles."
	}
	if !state.permissions.Has(discord.PermissionManageRoles) {
		return false, "Bot is missing the Manage Roles permission."
	}
	if role.Position >= state.topPosition {
		return false, "Bot cannot manage that role (role is above the bot)."
	}
	return true, ""
}

func (h *Handler) resolveBotRoleState(event *events.ApplicationCommandInteractionCreate, guildID snowflake.ID) (botRoleState, error) {
	caches := event.Client().Caches()
	selfMember, ok := caches.SelfMember(guildID)
	if !ok {
		selfID := event.Client().ApplicationID()
		if selfID == 0 {
			return botRoleState{}, errors.New("bot id not cached")
		}
		restMember, err := event.Client().Rest().GetMember(guildID, selfID)
		if err != nil {
			return botRoleState{}, err
		}
		selfMember = *restMember
	}

	state, missing := botRoleStateFromCache(caches, guildID, selfMember)
	if !missing {
		return state, nil
	}

	roles, err := event.Client().Rest().GetRoles(guildID)
	if err != nil {
		return state, err
	}

	return botRoleStateFromRoles(roles, guildID, selfMember.RoleIDs), nil
}

func botRoleStateFromCache(caches cache.Caches, guildID snowflake.ID, member discord.Member) (botRoleState, bool) {
	missing := false
	permissions := discord.Permissions(0)
	top := 0

	if role, ok := caches.Role(guildID, guildID); ok {
		permissions = role.Permissions
		top = role.Position
	} else {
		missing = true
	}

	for _, roleID := range member.RoleIDs {
		role, ok := caches.Role(guildID, roleID)
		if !ok {
			missing = true
			continue
		}
		permissions = permissions.Add(role.Permissions)
		if role.Position > top {
			top = role.Position
		}
		if permissions.Has(discord.PermissionAdministrator) {
			permissions = discord.PermissionsAll
		}
	}

	return botRoleState{
		permissions: permissions,
		topPosition: top,
	}, missing
}

func botRoleStateFromRoles(roles []discord.Role, guildID snowflake.ID, roleIDs []snowflake.ID) botRoleState {
	roleMap := make(map[snowflake.ID]discord.Role, len(roles))
	for _, role := range roles {
		roleMap[role.ID] = role
	}

	permissions := discord.Permissions(0)
	top := 0
	if role, ok := roleMap[guildID]; ok {
		permissions = role.Permissions
		top = role.Position
	}

	for _, roleID := range roleIDs {
		role, ok := roleMap[roleID]
		if !ok {
			continue
		}
		permissions = permissions.Add(role.Permissions)
		if role.Position > top {
			top = role.Position
		}
		if permissions.Has(discord.PermissionAdministrator) {
			permissions = discord.PermissionsAll
		}
	}

	return botRoleState{
		permissions: permissions,
		topPosition: top,
	}
}
