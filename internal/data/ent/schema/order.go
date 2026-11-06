package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Order holds the schema definition for a mall entity.
// All public ids come from IDMixin (UUIDv7); reference columns are plain
// strings to match the "no foreign keys" design of the source project.
type Order struct {
	ent.Schema
}

// Mixin of the Order.
func (Order) Mixin() []ent.Mixin {
	return []ent.Mixin{IDMixin{}, TimeMixin{}}
}

// Fields of the Order.
func (Order) Fields() []ent.Field {
	return []ent.Field{
		field.String("user_id").Default(""),
		field.String("order_sn").Unique(),
		field.Int32("status").Default(1),
		field.String("post").Default(""),
		field.Int64("total").Default(0),
		field.String("address").Default(""),
		field.String("name").Default(""),
		field.String("mobile").Default(""),
		field.Int64("add_time").Default(0),
		field.Int64("pay_time").Default(0),
	}
}

// Indexes of the Order.
func (Order) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("status", "add_time"),
	}
}

