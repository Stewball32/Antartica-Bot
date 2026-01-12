package schema

import "github.com/pocketbase/pocketbase/core"

func init() {
	Register(staticMessagesCollection)
}

func staticMessagesCollection() *core.Collection {
	collection := core.NewBaseCollection("static_messages")
	collection.Fields.Add(
		&core.TextField{Name: "guild_id", Required: true},
		&core.TextField{Name: "channel_id", Required: true},
		&core.TextField{Name: "message_id", Required: true},
		&core.SelectField{
			Name:     "type",
			Required: true,
			Values:   []string{"reaction_leaderboard", "role_toggles"},
		},
		&core.TextField{Name: "config"},
		&core.SelectField{
			Name:     "update",
			Required: true,
			Values:   []string{"instant", "hourly", "daily"},
		},
	)

	return collection
}
