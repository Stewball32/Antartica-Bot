package commands

import (
	"log/slog"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/snowflake/v2"
)

func RegisterCommands(client bot.Client, logger *slog.Logger, devGuildID *snowflake.ID) error {
	if logger == nil {
		logger = slog.Default()
	}

	cmds := All()
	if len(cmds) == 0 {
		logger.Warn("no commands registered")
		return nil
	}

	if devGuildID != nil {
		_, err := client.Rest().SetGuildCommands(client.ApplicationID(), *devGuildID, cmds)
		if err != nil {
			logger.Error("failed to register guild commands", slog.Any("err", err))
			return err
		}
		logger.Info("discord guild commands registered", slog.Int("count", len(cmds)), slog.String("guild_id", devGuildID.String()))
		return nil
	}

	_, err := client.Rest().SetGlobalCommands(client.ApplicationID(), cmds)
	if err != nil {
		logger.Error("failed to register commands", slog.Any("err", err))
		return err
	}
	logger.Info("discord commands registered", slog.Int("count", len(cmds)))
	return nil
}
