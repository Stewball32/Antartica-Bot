package stores

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"antartica-bot/internal/bus"

	"github.com/disgoorg/snowflake/v2"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type RoleToggleStore struct {
	app    core.App
	logger *slog.Logger
}

func NewRoleToggleStore(app core.App, logger *slog.Logger) *RoleToggleStore {
	if logger == nil && app != nil {
		logger = app.Logger()
	}
	return &RoleToggleStore{
		app:    app,
		logger: logger,
	}
}

func (s *RoleToggleStore) UpsertRoleToggle(ctx context.Context, guildID snowflake.ID, roleID snowflake.ID, permissions string, description *string) (bool, error) {
	if s == nil || s.app == nil {
		return false, errors.New("role toggle store is not configured")
	}

	records, err := s.app.FindAllRecords("role_toggles", dbx.HashExp{
		"guild_id": guildID.String(),
		"role_id":  roleID.String(),
	})
	if err != nil {
		return false, err
	}

	if len(records) > 0 {
		record := records[0]
		record.Set("permissions", permissions)
		if description != nil {
			record.Set("description", *description)
		}
		if err := s.app.SaveWithContext(ctx, record); err != nil {
			return false, err
		}
		return false, nil
	}

	collection, err := s.app.FindCollectionByNameOrId("role_toggles")
	if err != nil {
		return false, err
	}

	record := core.NewRecord(collection)
	record.Set("guild_id", guildID.String())
	record.Set("role_id", roleID.String())
	record.Set("permissions", permissions)
	if description != nil {
		record.Set("description", *description)
	}

	if err := s.app.SaveWithContext(ctx, record); err != nil {
		return false, err
	}

	if s.logger != nil {
		s.logger.Info(
			"role toggle added",
			slog.String("guild_id", guildID.String()),
			slog.String("role_id", roleID.String()),
		)
	}

	return true, nil
}

func (s *RoleToggleStore) RemoveRoleToggle(ctx context.Context, guildID snowflake.ID, roleID snowflake.ID) (int, error) {
	if s == nil || s.app == nil {
		return 0, errors.New("role toggle store is not configured")
	}

	records, err := s.app.FindAllRecords("role_toggles", dbx.HashExp{
		"guild_id": guildID.String(),
		"role_id":  roleID.String(),
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
			return deleted, fmt.Errorf("delete role toggle %s: %w", record.Id, err)
		}
		deleted++
	}

	if s.logger != nil {
		s.logger.Info(
			"role toggles removed",
			slog.String("guild_id", guildID.String()),
			slog.String("role_id", roleID.String()),
			slog.Int("count", deleted),
		)
	}

	return deleted, nil
}

func (s *RoleToggleStore) ListRoleToggles(ctx context.Context, guildID snowflake.ID) ([]bus.RoleToggle, error) {
	if s == nil || s.app == nil {
		return nil, errors.New("role toggle store is not configured")
	}

	records, err := s.app.FindAllRecords("role_toggles", dbx.HashExp{
		"guild_id": guildID.String(),
	})
	if err != nil {
		return nil, err
	}

	toggles := make([]bus.RoleToggle, 0, len(records))
	for _, record := range records {
		roleID, err := snowflake.Parse(record.GetString("role_id"))
		if err != nil {
			if s.logger != nil {
				s.logger.Warn("invalid role toggle role_id", slog.String("role_id", record.GetString("role_id")), slog.String("record_id", record.Id))
			}
			continue
		}

		toggles = append(toggles, bus.RoleToggle{
			RoleID:      roleID,
			Permissions: record.GetString("permissions"),
			Description: record.GetString("description"),
		})
	}

	return toggles, nil
}

var _ bus.RoleToggleStore = (*RoleToggleStore)(nil)
