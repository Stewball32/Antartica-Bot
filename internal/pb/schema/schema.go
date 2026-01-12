package schema

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/pocketbase/pocketbase/core"
)

type Builder func() *core.Collection

var builders []Builder

// Register adds a collection builder. Use one file per collection and register in init.
func Register(builder Builder) {
	if builder == nil {
		return
	}
	builders = append(builders, builder)
}

// Ensure creates missing collections at startup and adds missing fields to existing ones.
func Ensure(app core.App, logger *slog.Logger) error {
	if logger == nil {
		logger = app.Logger()
	}

	for _, build := range builders {
		collection := build()
		if collection == nil {
			continue
		}
		if collection.Name == "" {
			return errors.New("collection name is required")
		}

		existing, err := app.FindCollectionByNameOrId(collection.Name)
		if err == nil {
			added := 0
			updated := 0
			for _, field := range collection.Fields {
				existingField := existing.Fields.GetByName(field.GetName())
				if existingField == nil {
					existing.Fields.Add(field)
					added++
					continue
				}

				if existingField.Type() == core.FieldTypeSelect && field.Type() == core.FieldTypeSelect {
					existingSelect, okExisting := existingField.(*core.SelectField)
					newSelect, okNew := field.(*core.SelectField)
					if okExisting && okNew {
						if mergeSelectValues(existingSelect, newSelect.Values) {
							updated++
						}
					}
				}
			}
			if added > 0 || updated > 0 {
				if err := app.Save(existing); err != nil {
					return fmt.Errorf("update collection %q: %w", collection.Name, err)
				}
				logger.Info(
					"collection updated",
					slog.String("collection", collection.Name),
					slog.Int("fields_added", added),
					slog.Int("fields_updated", updated),
				)
			}
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		if err := app.Save(collection); err != nil {
			return fmt.Errorf("create collection %q: %w", collection.Name, err)
		}
		logger.Info("collection created", slog.String("collection", collection.Name))
	}

	return nil
}

func mergeSelectValues(field *core.SelectField, values []string) bool {
	if field == nil || len(values) == 0 {
		return false
	}

	existing := make(map[string]struct{}, len(field.Values))
	for _, value := range field.Values {
		existing[value] = struct{}{}
	}

	updated := false
	for _, value := range values {
		if _, ok := existing[value]; ok {
			continue
		}
		field.Values = append(field.Values, value)
		existing[value] = struct{}{}
		updated = true
	}

	return updated
}
