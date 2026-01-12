package schema

import "github.com/pocketbase/pocketbase/core"

func init() {
	Register(roleTogglesCollection)
}

func roleTogglesCollection() *core.Collection {
	collection := core.NewBaseCollection("role_toggles")
	collection.Fields.Add(
		&core.TextField{Name: "guild_id", Required: true},
		&core.TextField{Name: "role_id", Required: true},
		&core.TextField{Name: "permissions", Required: true},
		&core.TextField{Name: "description"},
	)

	return collection
}
