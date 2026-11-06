package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Inventory holds the schema definition for a mall entity.
// All public ids come from IDMixin (UUIDv7); reference columns are plain
// strings to match the "no foreign keys" design of the source project.
type Inventory struct {
	ent.Schema
}

// Mixin of the Inventory.
func (Inventory) Mixin() []ent.Mixin {
	return []ent.Mixin{IDMixin{}, TimeMixin{}}
}

// Fields of the Inventory.
func (Inventory) Fields() []ent.Field {
	return []ent.Field{
		field.String("sku_id").Unique(),
		field.Int64("total").Default(0),
		field.Int64("locked").Default(0),
		field.Int64("sold").Default(0),
		field.Int64("version").Default(0),
	}
}

