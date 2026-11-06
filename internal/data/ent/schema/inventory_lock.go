package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// InventoryLock holds the schema definition for a mall entity.
// All public ids come from IDMixin (UUIDv7); reference columns are plain
// strings to match the "no foreign keys" design of the source project.
type InventoryLock struct {
	ent.Schema
}

// Mixin of the InventoryLock.
func (InventoryLock) Mixin() []ent.Mixin {
	return []ent.Mixin{IDMixin{}, TimeMixin{}}
}

// Fields of the InventoryLock.
func (InventoryLock) Fields() []ent.Field {
	return []ent.Field{
		field.String("order_sn").Default(""),
		field.String("sku_id").Default(""),
		field.Int32("num").Default(0),
		field.Int32("status").Default(1),
	}
}

// Indexes of the InventoryLock.
func (InventoryLock) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_sn", "sku_id").Unique(),
	}
}

