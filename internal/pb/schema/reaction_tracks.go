package schema

import "github.com/pocketbase/pocketbase/core"

func init() {
	Register(reactionTracksCollection)
}

func reactionTracksCollection() *core.Collection {
	collection := core.NewBaseCollection("reaction_tracks")
	collection.Fields.Add(
		&core.TextField{Name: "guild_id", Required: true},
		&core.TextField{Name: "emoji_id"},
		&core.TextField{Name: "emoji_name"},
		&core.TextField{Name: "title", Required: true},
		&core.TextField{Name: "description"},
	)

	return collection
}
