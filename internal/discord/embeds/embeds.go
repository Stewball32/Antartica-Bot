package embeds

import (
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
)

type EmbedTone string

const (
	EmbedInfo     EmbedTone = "info"
	EmbedSuccess  EmbedTone = "success"
	EmbedDecline  EmbedTone = "decline"
	EmbedQuestion EmbedTone = "question"
	EmbedError    EmbedTone = "error"
	EmbedWarn     EmbedTone = "warn"
	EmbedDebug    EmbedTone = "debug"
	EmbedNeutral  EmbedTone = "neutral"
)

const (
	embedInfoColor     = 0x3B82F6
	embedSuccessColor  = 0x22C55E
	embedDeclineColor  = 0xDC2626
	embedQuestionColor = 0x06B6D4
	embedErrorColor    = 0xEF4444
	embedWarnColor     = 0xF59E0B
	embedDebugColor    = 0x6B7280
	embedNeutralColor  = 0x9CA3AF
)

type EmbedTemplate struct {
	Tone        EmbedTone
	Title       string
	Description string
	Fields      []discord.EmbedField
	Footer      string
	Timestamp   *time.Time
}

// EmbedColor returns a themed color. Defaults to info when tone is empty or unknown.
func EmbedColor(tone EmbedTone) int {
	switch strings.ToLower(string(tone)) {
	case string(EmbedSuccess):
		return embedSuccessColor
	case string(EmbedDecline):
		return embedDeclineColor
	case string(EmbedQuestion):
		return embedQuestionColor
	case string(EmbedError):
		return embedErrorColor
	case string(EmbedWarn):
		return embedWarnColor
	case string(EmbedDebug):
		return embedDebugColor
	case string(EmbedNeutral):
		return embedNeutralColor
	case string(EmbedInfo):
		fallthrough
	default:
		return embedInfoColor
	}
}

// BuildEmbed creates a standardized embed from a template.
func BuildEmbed(template EmbedTemplate) discord.Embed {
	title := strings.TrimSpace(template.Title)
	if title == "" {
		title = defaultToneTitle(template.Tone)
	}

	embed := discord.Embed{
		Title:       title,
		Description: strings.TrimSpace(template.Description),
		Color:       EmbedColor(template.Tone),
	}

	if len(template.Fields) > 0 {
		embed.Fields = template.Fields
	}

	if footer := strings.TrimSpace(template.Footer); footer != "" {
		embed.Footer = &discord.EmbedFooter{Text: footer}
	}

	if template.Timestamp != nil {
		embed.Timestamp = template.Timestamp
	}

	return embed
}

func defaultToneTitle(tone EmbedTone) string {
	switch strings.ToLower(string(tone)) {
	case string(EmbedSuccess):
		return "Success"
	case string(EmbedDecline):
		return "Declined"
	case string(EmbedQuestion):
		return "Question"
	case string(EmbedError):
		return "Error"
	case string(EmbedWarn):
		return "Warning"
	case string(EmbedDebug):
		return "Debug"
	case string(EmbedNeutral):
		return "Note"
	case string(EmbedInfo):
		fallthrough
	default:
		return "Info"
	}
}
