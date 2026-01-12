package bus

import (
	"context"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

const DefaultBuffer = 128

type Bus struct {
	DiscordEvents  chan DiscordEvent
	DiscordActions chan DiscordAction
}

func New(buffer int) *Bus {
	if buffer <= 0 {
		buffer = DefaultBuffer
	}

	return &Bus{
		DiscordEvents:  make(chan DiscordEvent, buffer),
		DiscordActions: make(chan DiscordAction, buffer),
	}
}

type DiscordEvent interface {
	discordEvent()
}

type DiscordAction interface {
	discordAction()
}

type ReactionAdded struct {
	GuildID   snowflake.ID
	ChannelID snowflake.ID
	MessageID snowflake.ID
	UserID    snowflake.ID
	AuthorID  snowflake.ID
	EmojiName string
	EmojiID   *snowflake.ID
}

func (ReactionAdded) discordEvent() {}

type ReactionRemoved struct {
	GuildID   snowflake.ID
	ChannelID snowflake.ID
	MessageID snowflake.ID
	UserID    snowflake.ID
	AuthorID  snowflake.ID
	EmojiName string
	EmojiID   *snowflake.ID
}

func (ReactionRemoved) discordEvent() {}

type ReactionRemovedEmoji struct {
	GuildID   snowflake.ID
	ChannelID snowflake.ID
	MessageID snowflake.ID
	EmojiName string
	EmojiID   *snowflake.ID
}

func (ReactionRemovedEmoji) discordEvent() {}

type ReactionRemovedAll struct {
	GuildID   snowflake.ID
	ChannelID snowflake.ID
	MessageID snowflake.ID
}

func (ReactionRemovedAll) discordEvent() {}

type MessageDeleted struct {
	GuildID   snowflake.ID
	ChannelID snowflake.ID
	MessageID snowflake.ID
}

func (MessageDeleted) discordEvent() {}

type InteractionReceived struct {
	InteractionID   snowflake.ID
	InteractionType discord.InteractionType
	GuildID         *snowflake.ID
	ChannelID       snowflake.ID
	UserID          snowflake.ID
}

func (InteractionReceived) discordEvent() {}

type SendMessage struct {
	ChannelID snowflake.ID
	Content   string
}

func (SendMessage) discordAction() {}

type EditMessage struct {
	ChannelID snowflake.ID
	MessageID snowflake.ID
	Content   string
	Embeds    []discord.Embed
}

func (EditMessage) discordAction() {}

type LogLevel string

const (
	LogDebug LogLevel = "debug"
	LogInfo  LogLevel = "info"
	LogWarn  LogLevel = "warn"
	LogError LogLevel = "error"
)

type LogField struct {
	Name   string
	Value  string
	Inline bool
}

type LogEvent struct {
	GuildID     snowflake.ID
	Category    string
	Level       LogLevel
	Title       string
	Description string
	Fields      []LogField
	Timestamp   time.Time
}

func (LogEvent) discordAction() {}

type ReactionTrack struct {
	EmojiID     string
	EmojiName   string
	Title       string
	Description string
}

type RoleToggle struct {
	RoleID      snowflake.ID
	Permissions string
	Description string
}

type RoleToggleStore interface {
	UpsertRoleToggle(ctx context.Context, guildID snowflake.ID, roleID snowflake.ID, permissions string, description *string) (bool, error)
	RemoveRoleToggle(ctx context.Context, guildID snowflake.ID, roleID snowflake.ID) (int, error)
	ListRoleToggles(ctx context.Context, guildID snowflake.ID) ([]RoleToggle, error)
}

type ReactionTrackStore interface {
	UpsertReactionTrack(ctx context.Context, guildID snowflake.ID, emojiID string, emojiName string, title string, description string) (bool, error)
	RemoveReactionTrack(ctx context.Context, guildID snowflake.ID, emojiID string, emojiName string) (int, error)
	ListReactionTracks(ctx context.Context, guildID snowflake.ID) ([]ReactionTrack, error)
}

type StaticMessage struct {
	ChannelID snowflake.ID
	MessageID snowflake.ID
	Type      string
	Config    string
	Update    string
}

type StaticMessageStore interface {
	CreateStaticMessage(ctx context.Context, guildID snowflake.ID, channelID snowflake.ID, messageID snowflake.ID, messageType string, config string, update string) error
	RemoveStaticMessage(ctx context.Context, guildID snowflake.ID, messageID snowflake.ID) (int, error)
	ListStaticMessages(ctx context.Context, guildID snowflake.ID) ([]StaticMessage, error)
}
