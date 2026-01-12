package schema

import "github.com/pocketbase/pocketbase/core"

func init() {
	Register(reactionRecordsCollection)
}

func reactionRecordsCollection() *core.Collection {
	collection := core.NewBaseCollection("reaction_records")
	collection.Fields.Add(
		&core.TextField{Name: "guild_id", Required: true},
		&core.TextField{Name: "message_id", Required: true},
		&core.TextField{Name: "user_id", Required: true},
		&core.TextField{Name: "emoji_id"},
		&core.TextField{Name: "emoji_name"},
		&core.NumberField{Name: "reactions", Required: true},
	)

	return collection
}
