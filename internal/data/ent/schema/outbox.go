package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// OrderEventOutbox holds the schema definition for a mall entity.
// All public ids come from IDMixin (UUIDv7); reference columns are plain
// strings to match the "no foreign keys" design of the source project.
type OrderEventOutbox struct {
	ent.Schema
}

// Mixin of the OrderEventOutbox.
func (OrderEventOutbox) Mixin() []ent.Mixin {
	return []ent.Mixin{IDMixin{}, TimeMixin{}}
}

// Fields of the OrderEventOutbox.
func (OrderEventOutbox) Fields() []ent.Field {
	return []ent.Field{
		field.String("event_id").Unique(),
		field.String("event_type").Default(""),
		field.String("order_sn").Default(""),
		field.Text("payload").Default(""),
		field.Int32("status").Default(0),
		field.Int32("retry_count").Default(0),
	}
}

// Indexes of the OrderEventOutbox.
func (OrderEventOutbox) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status", "retry_count"),
	}
}

