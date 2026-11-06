package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ConsumedEvent holds the schema definition for a mall entity.
// All public ids come from IDMixin (UUIDv7); reference columns are plain
// strings to match the "no foreign keys" design of the source project.
type ConsumedEvent struct {
	ent.Schema
}

// Mixin of the ConsumedEvent.
func (ConsumedEvent) Mixin() []ent.Mixin {
	return []ent.Mixin{IDMixin{}, TimeMixin{}}
}

// Fields of the ConsumedEvent.
func (ConsumedEvent) Fields() []ent.Field {
	return []ent.Field{
		field.String("event_id").Default(""),
		field.String("consumer").Default(""),
	}
}

// Indexes of the ConsumedEvent.
func (ConsumedEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("event_id", "consumer").Unique(),
	}
}

