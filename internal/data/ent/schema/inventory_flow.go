package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// InventoryFlow holds the schema definition for a mall entity.
// All public ids come from IDMixin (UUIDv7); reference columns are plain
// strings to match the "no foreign keys" design of the source project.
type InventoryFlow struct {
	ent.Schema
}

// Mixin of the InventoryFlow.
func (InventoryFlow) Mixin() []ent.Mixin {
	return []ent.Mixin{IDMixin{}, TimeMixin{}}
}

// Fields of the InventoryFlow.
func (InventoryFlow) Fields() []ent.Field {
	return []ent.Field{
		field.String("sku_id").Default(""),
		field.String("order_sn").Default(""),
		field.Int32("num").Default(0),
		field.Int32("flow_type").Default(0),
	}
}

// Indexes of the InventoryFlow.
func (InventoryFlow) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("sku_id"),
		index.Fields("order_sn"),
	}
}

