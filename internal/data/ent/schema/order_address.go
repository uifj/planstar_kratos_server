package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// OrderAddress holds the schema definition for a mall entity.
// All public ids come from IDMixin (UUIDv7); reference columns are plain
// strings to match the "no foreign keys" design of the source project.
type OrderAddress struct {
	ent.Schema
}

// Mixin of the OrderAddress.
func (OrderAddress) Mixin() []ent.Mixin {
	return []ent.Mixin{IDMixin{}, TimeMixin{}}
}

// Fields of the OrderAddress.
func (OrderAddress) Fields() []ent.Field {
	return []ent.Field{
		field.String("order_id").Default(""),
		field.String("name").Default(""),
		field.String("mobile").Default(""),
		field.String("province").Default(""),
		field.String("city").Default(""),
		field.String("districts").Default(""),
		field.String("address").Default(""),
		field.String("post_code").Default(""),
	}
}

// Indexes of the OrderAddress.
func (OrderAddress) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id"),
	}
}

