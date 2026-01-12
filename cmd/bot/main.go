package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"

	"antartica-bot/internal/bus"
	"antartica-bot/internal/config"
	discordbridge "antartica-bot/internal/discord"
	discordactions "antartica-bot/internal/discord/actions"
	"antartica-bot/internal/discord/commands"
	pbconsumers "antartica-bot/internal/pb/consumers"
	pbhooks "antartica-bot/internal/pb/hooks"
	pbstores "antartica-bot/internal/pb/stores"

	"github.com/disgoorg/snowflake/v2"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	logger := slog.Default()
	app := pocketbase.New()
	eventBus := bus.New(bus.DefaultBuffer)

	pbhooks.RegisterHooks(app, eventBus, logger)

	args := os.Args[1:]
	if shouldDefaultServe(args) {
		app.RootCmd.SetArgs([]string{"serve"})
	}

	if shouldStartDiscord(args) {
		cfg, err := readConfig(logger)
		if err != nil {
			if errors.Is(err, errConfigCreated) {
				os.Exit(1)
			}
			logger.Error("config load failed", slog.Any("err", err))
			os.Exit(1)
		}

		token := strings.TrimSpace(cfg.Discord.Token)
		if token == "" {
			logger.Error("discord token missing from config")
			os.Exit(1)
		}

		var devGuildID *snowflake.ID
		if cfg.Dev.Enabled {
			rawGuildID := strings.TrimSpace(cfg.Dev.GuildID)
			if rawGuildID == "" {
				logger.Error("dev guild_id missing from config")
				os.Exit(1)
			}
			parsed, err := snowflake.Parse(rawGuildID)
			if err != nil {
				logger.Error("dev guild_id invalid", slog.Any("err", err))
				os.Exit(1)
			}
			devGuildID = &parsed
		}

		botConfig := discordbridge.DefaultConfig()
		botConfig.Token = token

		roleToggleStore := pbstores.NewRoleToggleStore(app, logger)
		reactionTrackStore := pbstores.NewReactionTrackStore(app, logger)
		staticMessageStore := pbstores.NewStaticMessageStore(app, logger)
		discordBot, err := discordbridge.New(botConfig, eventBus, logger, roleToggleStore, reactionTrackStore, staticMessageStore)
		if err != nil {
			logger.Error("failed to create discord client", slog.Any("err", err))
			os.Exit(1)
		}

		ctx, cancel := context.WithCancel(context.Background())

		app.OnServe().BindFunc(func(e *core.ServeEvent) error {
			if err := commands.RegisterCommands(discordBot.Client(), logger, devGuildID); err != nil {
				logger.Error("command registration failed", slog.Any("err", err))
			}
			if err := discordBot.Start(context.Background()); err != nil {
				return err
			}

			pbconsumers.StartDiscordConsumer(ctx, app, eventBus, logger)
			discordactions.StartActionWorker(ctx, discordBot.Client(), eventBus, logger)

			return e.Next()
		})

		app.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
			cancel()
			discordBot.Close(context.Background())
			return e.Next()
		})
	}

	if err := app.Start(); err != nil {
		logger.Error("pocketbase exited", slog.Any("err", err))
		os.Exit(1)
	}
}

var errConfigCreated = errors.New("config.yaml created")

func readConfig(logger *slog.Logger) (config.Config, error) {
	const configFile = "config.yaml"
	cfg, err := config.Load(configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return cfg, err
		}

		if logger == nil {
			logger = slog.Default()
		}
		if err := createConfigFromExample(configFile); err != nil {
			return cfg, err
		}

		logger.Info("created config.yaml from embedded config.example.yaml")
		logger.Info("fill in discord.client_id, discord.secret, discord.token, and dev.guild_id if dev.enabled")
		return cfg, errConfigCreated
	}
	return cfg, nil
}

func createConfigFromExample(destination string) error {
	if len(config.DefaultConfigYAML) == 0 {
		return errors.New("embedded config template missing")
	}
	return os.WriteFile(destination, config.DefaultConfigYAML, 0644)
}

func shouldStartDiscord(args []string) bool {
	if cmd := firstNonFlag(args); cmd != "" {
		return cmd == "serve"
	}

	if hasAnyFlag(args, "-h", "--help", "-v", "--version") {
		return false
	}

	return true
}

func shouldDefaultServe(args []string) bool {
	if cmd := firstNonFlag(args); cmd != "" {
		return false
	}

	return !hasAnyFlag(args, "-h", "--help", "-v", "--version")
}

func firstNonFlag(args []string) string {
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		return arg
	}
	return ""
}

func hasAnyFlag(args []string, flags ...string) bool {
	if len(args) == 0 {
		return false
	}

	set := make(map[string]struct{}, len(flags))
	for _, flag := range flags {
		set[flag] = struct{}{}
	}

	for _, arg := range args {
		if _, ok := set[arg]; ok {
			return true
		}
	}

	return false
}
