package discord

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"antartica-bot/internal/bus"
	"antartica-bot/internal/discord/handlers"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
)

type Config struct {
	Token   string
	Intents gateway.Intents
}

func DefaultConfig() Config {
	return Config{
		Intents: gateway.IntentsNonPrivileged,
	}
}

type Bot struct {
	client  bot.Client
	handler *handlers.Handler
}

func New(cfg Config, eventBus *bus.Bus, logger *slog.Logger, roleToggleStore bus.RoleToggleStore, reactionTrackStore bus.ReactionTrackStore, staticMessageStore bus.StaticMessageStore) (*Bot, error) {
	if strings.TrimSpace(cfg.Token) == "" {
		return nil, errors.New("discord token is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.Intents == 0 {
		cfg.Intents = DefaultConfig().Intents
	}

	var handler *handlers.Handler
	client, err := disgo.New(cfg.Token,
		bot.WithLogger(logger),
		bot.WithGatewayConfigOpts(gateway.WithIntents(cfg.Intents)),
		bot.WithEventListenerFunc(func(event *events.GuildMessageReactionAdd) {
			if handler != nil {
				handler.OnGuildMessageReactionAdd(event)
			}
		}),
		bot.WithEventListenerFunc(func(event *events.GuildMessageReactionRemove) {
			if handler != nil {
				handler.OnGuildMessageReactionRemove(event)
			}
		}),
		bot.WithEventListenerFunc(func(event *events.GuildMessageReactionRemoveEmoji) {
			if handler != nil {
				handler.OnGuildMessageReactionRemoveEmoji(event)
			}
		}),
		bot.WithEventListenerFunc(func(event *events.GuildMessageReactionRemoveAll) {
			if handler != nil {
				handler.OnGuildMessageReactionRemoveAll(event)
			}
		}),
		bot.WithEventListenerFunc(func(event *events.GuildMessageDelete) {
			if handler != nil {
				handler.OnGuildMessageDelete(event)
			}
		}),
		bot.WithEventListenerFunc(func(event *events.InteractionCreate) {
			if handler != nil {
				handler.OnInteractionCreate(event)
			}
		}),
		bot.WithEventListenerFunc(func(event *events.ApplicationCommandInteractionCreate) {
			if handler != nil {
				handler.OnApplicationCommand(event)
			}
		}),
	)
	if err != nil {
		return nil, err
	}

	handler = handlers.New(client, eventBus, logger, roleToggleStore, reactionTrackStore, staticMessageStore)

	return &Bot{
		client:  client,
		handler: handler,
	}, nil
}

func (b *Bot) Start(ctx context.Context) error {
	return b.client.OpenGateway(ctx)
}

func (b *Bot) Close(ctx context.Context) {
	b.client.Close(ctx)
}

func (b *Bot) Client() bot.Client {
	return b.client
}
