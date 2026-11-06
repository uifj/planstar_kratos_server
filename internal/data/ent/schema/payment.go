package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Payment holds the schema definition for a mall entity.
// All public ids come from IDMixin (UUIDv7); reference columns are plain
// strings to match the "no foreign keys" design of the source project.
type Payment struct {
	ent.Schema
}

// Mixin of the Payment.
func (Payment) Mixin() []ent.Mixin {
	return []ent.Mixin{IDMixin{}, TimeMixin{}}
}

// Fields of the Payment.
func (Payment) Fields() []ent.Field {
	return []ent.Field{
		field.String("payment_no").Unique(),
		field.String("order_sn").Default(""),
		field.String("user_id").Default(""),
		field.Int64("amount").Default(0),
		field.String("channel").Default(""),
		field.Int32("status").Default(1),
	}
}

// Indexes of the Payment.
func (Payment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_sn"),
		index.Fields("user_id"),
	}
}

