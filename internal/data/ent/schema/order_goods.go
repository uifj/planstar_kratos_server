package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// OrderGoods holds the schema definition for a mall entity.
// All public ids come from IDMixin (UUIDv7); reference columns are plain
// strings to match the "no foreign keys" design of the source project.
type OrderGoods struct {
	ent.Schema
}

// Mixin of the OrderGoods.
func (OrderGoods) Mixin() []ent.Mixin {
	return []ent.Mixin{IDMixin{}, TimeMixin{}}
}

// Fields of the OrderGoods.
func (OrderGoods) Fields() []ent.Field {
	return []ent.Field{
		field.String("order_id").Default(""),
		field.String("sku_id").Default(""),
		field.String("sku_name").Default(""),
		field.Int64("sku_price").Default(0),
		field.Int32("num").Default(0),
		field.Int64("total_price").Default(0),
	}
}

// Indexes of the OrderGoods.
func (OrderGoods) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id"),
		index.Fields("sku_id"),
	}
}

