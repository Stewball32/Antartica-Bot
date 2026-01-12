# Antarctica Bot

Single-binary Discord bot built with Disgo and PocketBase. It wires Discord events to PocketBase (and vice versa) through an in-process bus, so you can store state, react to events, and push updates back to Discord.

## Features

- Reaction tracking with leaderboard messages that auto-update.
- Self-assignable roles with optional permission gating.
- Static message management (role lists and reaction leaderboards).
- Embedded PocketBase for storage and admin UI.

## Quick start

1) Run once to generate `config.yaml`, then add your Discord credentials.

```bash
go run ./cmd/bot
```

2) Update `config.yaml` (Discord tokens + optional PocketBase port).

3) Run the bot again (PocketBase starts with `serve` by default).

```bash
go run ./cmd/bot
```

PocketBase commands still work when you build a binary:

```bash
go build -o bot ./cmd/bot
./bot superuser
./bot serve --dir ./pb_data
```
If `pocketbase.port` is set in `config.yaml`, the bot will pass `--http=127.0.0.1:<port>` automatically unless you explicitly provide `--http`.

## Slash commands

- `/reaction` admin tools for tracking emojis and leaderboard messages.
- `/role` admin tools for self-assignable roles and role list messages.
- `/toggle-role` user command to self-assign roles.

## Data model

Collections are created/updated automatically on boot:

- `reaction_tracks` and `reaction_records` for tracking emoji reactions.
- `reaction_leaderboard` for leaderboard aggregates.
- `role_toggles` for self-assignable roles.
- `static_messages` for managed embeds (role lists and leaderboards).

## Project layout

- `cmd/bot/main.go`: boots PocketBase + Disgo, lifecycle wiring
- `internal/bus/bus.go`: internal event bus and event/action types
- `internal/discord/`: Disgo client + handlers/actions/embeds subpackages
- `internal/pb/hooks/`: PocketBase hooks
- `internal/pb/consumers/`: Discord event consumer + reaction processor
- `internal/pb/messages/`: Static message builders and updaters
- `internal/pb/stores/`: PocketBase stores
- `internal/pb/schema/`: PocketBase collection definitions

## Where to add logic

- Discord to PocketBase: `internal/pb/consumers/discord_consumer.go`
- PocketBase to Discord: `internal/pb/hooks/hooks.go` + `internal/discord/actions/actions.go`

## Notes

- Default gateway intents are `gateway.IntentsNonPrivileged`. Add `gateway.IntentMessageContent` if you need raw message content.
- The bot only starts when running the `serve` command (or no command at all).
- `config.yaml` is generated from an embedded template (`internal/config/config.example.yaml`) if it doesn't exist.
